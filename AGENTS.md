# Science Swap

Science Swap helps people find a partner at another university for a semester exchange. The core workflow described in `specs/science_swap.txt` is mutual matching by field of study, overlapping availability, and preferred regions.

## Project root

- Keep all application source under `/home/hoka416/my_project`. Paths in this file are relative to that directory.

## Product requirements

- Use `specs/science_swap.txt` for technical requirements and `docs/PRODUCT.md` for the agreed MVP scope. `specs/spec.txt` describes a broader concept.
- Keep participation open to everyone regardless of academic role. Support multiple preferred regions and email contact after a mutual match. Let each participant decide what to share at that point, including their own email address.
- Use the `SSwap` prefix for application tables.

## Backend and database

- Build the backend in Go with the standard library's `net/http`. Use `pgx` for PostgreSQL access and `goose` for migrations.
- Use PostgreSQL 17 or newer. Read the connection string from `DATABASE_URL`. Keep local credentials in `.env`, exclude it from version control, and commit an `.env.example` with placeholders.
- Enable these extensions in the first migration, with `cube` before `earthdistance`.
  - `pg_trgm` for fuzzy university-name search, such as “FU Berlin”.
  - `unaccent` for search that ignores accents and umlauts.
  - `btree_gist` for GiST indexes or constraints combining an ID and a date range.
  - `citext` for case-insensitive email addresses.
  - `cube` and `earthdistance` for future coordinate-based radius search.
- Store availability and other date periods as `daterange` and test overlap with PostgreSQL's `&&` operator.
- Use `faculty_swap_dev` for development and `faculty_swap_test` for tests. Run database tests against the real PostgreSQL test database. The brief names Postgres.app for macOS. In this Ubuntu WSL workspace, use the local Homebrew PostgreSQL 17 instance.

## Frontend and localization

- Put the Vue 3, TypeScript, Vite, and Tailwind CSS frontend in `web/`. Build the frontend before compiling the Go binary and include the build output using `go:embed`.
- Use `vue-i18n` for all user-facing strings. Keep German and English translations in `web/src/locales/de.json` and `web/src/locales/en.json`. Add a test that compares their full translation key sets.
- Choose the initial language from the browser preferences, provide a header language switcher, and remember the user's selection.
- Use `Intl.DisplayNames` for country names and `Intl.DateTimeFormat` for dates. Use ISCED-F fields with both German and English labels.

## Build, deployment, and workflow

- Provide `make dev`, `make test`, `make build`, and `make db-migrate` when scaffolding the app.
- Use a multi-stage Dockerfile to build and package a single Go binary in one container. Target Fly.io's `fra` region for deployment.
- Use English for database names, tables, columns, types, structs, functions, variables, packages, filenames, API paths, comments, and commit messages.
- Work in small steps. Once Git is initialized, commit each completed step after its tests pass. Ask before adding dependencies beyond those already specified in the project brief.
