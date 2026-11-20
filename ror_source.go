package main

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/stdlib"
)

const rorLatestRecord = "https://zenodo.org/api/records/6347574"

var rorReleaseArchive = regexp.MustCompile(`^(v[0-9]+\.[0-9]+)-[0-9]{4}-[0-9]{2}-[0-9]{2}-ror-data\.zip$`)

func importRORCommand(ctx context.Context, db *sql.DB, args []string) error {
	flags := flag.NewFlagSet("import-ror", flag.ContinueOnError)
	latest := flags.Bool("latest", false, "download the latest full ROR release from Zenodo")
	file := flags.String("file", "", "path to a full, named ROR release ZIP")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *latest == (*file != "") {
		return errors.New("import-ror requires exactly one of --latest or --file PATH")
	}

	archivePath := *file
	if *latest {
		var err error
		var cleanup func()
		archivePath, cleanup, err = downloadROR(ctx)
		if err != nil {
			return err
		}
		defer cleanup()
	}
	release, reader, closeArchive, err := openRORArchive(archivePath)
	if err != nil {
		return err
	}
	defer closeArchive()

	connection, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("connect for ROR import: %w", err)
	}
	defer connection.Close()
	return connection.Raw(func(driverConn any) error {
		result, err := importROR(ctx, driverConn.(*stdlib.Conn).Conn(), reader, release)
		if err != nil {
			return err
		}
		log.Printf("ROR %s: %d new, %d updated, %d deactivated", release, result.added, result.updated, result.deactivated)
		return nil
	})
}

func openRORArchive(path string) (string, io.Reader, func(), error) {
	name := filepath.Base(path)
	release := rorReleaseArchive.FindStringSubmatch(name)
	if release == nil {
		return "", nil, nil, errors.New("expected a ROR archive named vX.Y-YYYY-MM-DD-ror-data.zip")
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		return "", nil, nil, fmt.Errorf("open ROR archive: %w", err)
	}
	data, err := archive.Open(strings.TrimSuffix(name, ".zip") + ".json")
	if err != nil {
		archive.Close()
		return "", nil, nil, fmt.Errorf("open ROR JSON in archive: %w", err)
	}
	return release[1], data, func() {
		data.Close()
		archive.Close()
	}, nil
}

func downloadROR(ctx context.Context) (string, func(), error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	metadata, err := getRORResponse(ctx, client, rorLatestRecord)
	if err != nil {
		return "", nil, err
	}
	defer metadata.Body.Close()
	var record struct {
		Files []struct {
			Key      string
			Size     int64
			Checksum string
			Links    struct{ Self string }
		}
	}
	if err := json.NewDecoder(io.LimitReader(metadata.Body, 2<<20)).Decode(&record); err != nil {
		return "", nil, fmt.Errorf("decode ROR release metadata: %w", err)
	}
	for _, file := range record.Files {
		if !rorReleaseArchive.MatchString(file.Key) {
			continue
		}
		return fetchRORArchive(ctx, client, file.Key, file.Links.Self, file.Checksum, file.Size)
	}
	return "", nil, errors.New("ROR release metadata contains no supported data ZIP")
}

func getRORResponse(ctx context.Context, client *http.Client, address string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch ROR data: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("fetch ROR data: %s", resp.Status)
	}
	return resp, nil
}

func fetchRORArchive(ctx context.Context, client *http.Client, name, address, checksum string, size int64) (string, func(), error) {
	link, err := url.Parse(address)
	if err != nil || link.Scheme != "https" || link.Host != "zenodo.org" {
		return "", nil, errors.New("invalid ROR file URL in Zenodo metadata")
	}
	hashAlgorithm, hashText, _ := strings.Cut(checksum, ":")
	expected, err := hex.DecodeString(hashText)
	if err != nil || hashAlgorithm != "md5" || len(expected) != md5.Size || size <= 0 || size > 1<<30 {
		return "", nil, errors.New("invalid ROR file size or checksum in Zenodo metadata")
	}

	resp, err := getRORResponse(ctx, client, address)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	dir, err := os.MkdirTemp("", "science-swap-ror-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	path := filepath.Join(dir, name)
	out, err := os.Create(path)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	hash := md5.New() // Matches Zenodo's published file checksum.
	count, copyErr := io.Copy(io.MultiWriter(out, hash), io.LimitReader(resp.Body, size+1))
	if err := errors.Join(copyErr, out.Close()); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("download ROR archive: %w", err)
	}
	if count != size || !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), hashText) {
		cleanup()
		return "", nil, errors.New("ROR download does not match the published size and checksum")
	}
	return path, cleanup, nil
}
