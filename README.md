# Science Swap

Science Swap is a web app for finding a partner at another university for a semester exchange. The planned matching workflow uses academic field, overlapping availability, and preferred regions. An exchange match requires interest from both people.

The starter app displays the version of the PostgreSQL server it is connected to. It serves the Vue interface and API from a single Go binary.

## Local development

Use Go 1.26 or newer, Node.js 24, GNU Make, and PostgreSQL 17 or newer.

Create `faculty_swap_dev` and `faculty_swap_test` with `createdb` if they are missing, then copy `.env.example` to `.env` and set both connection strings. With the local Homebrew server and Unix-socket peer authentication, use these values:

```sh
DATABASE_URL='postgres:///faculty_swap_dev?host=/tmp'
TEST_DATABASE_URL='postgres:///faculty_swap_test?host=/tmp'
```

| Command | What it does |
| --- | --- |
| `make db-migrate` | Applies Goose migrations to `DATABASE_URL` |
| `make dev` | Builds and serves the app at `http://127.0.0.1:8080` |
| `make test` | Checks translation keys and exercises the API and migrations against `faculty_swap_test` |
| `make build` | Writes the self-contained app to `bin/science-swap` |

## Reference data

`make db-migrate` creates the institutions table and seeds the bilingual ISCED-F fields and the EU and DACH region groups. To import university records into the local development database, load `.env` and use the built binary:

```sh
set -a
. ./.env
set +a
./bin/science-swap import-ror --latest
# Or use a complete release ZIP with its original filename.
./bin/science-swap import-ror --file data/ror/v2.16-2026-11-19-ror-data.zip
```

Each import reconciles the database with a complete ROR snapshot. It preserves institution IDs and marks existing records that leave the active education set. Downloads and archives under `data/` are Git-ignored.

## Deployment

The multi-stage Dockerfile uses Node.js 24 to build Vue, then embeds the result and migrations in the Go binary. `fly.toml` targets Frankfurt (`fra`) and runs migrations before release. The Fly app needs a PostgreSQL 17+ connection in its `DATABASE_URL` secret.

For a new Fly Managed Postgres database, enable `cube` and then `earthdistance` through Fly's administrator API before the first deploy. The app database role cannot create `earthdistance`.

```sh
flyctl mpg databases extensions enable cube --cluster "$FLY_MPG_CLUSTER" --database fly-db
flyctl mpg databases extensions enable earthdistance --cluster "$FLY_MPG_CLUSTER" --database fly-db
```

Pushes to `main` trigger the [GitHub Actions deployment workflow](.github/workflows/deploy.yml). It runs `make test` against PostgreSQL 17, deploys to Fly.io after the tests pass, and checks that the live API can read the database version.

The workflow uses the repository secret `FLY_API_TOKEN`. Rotate it before expiry using an app-scoped token from `flyctl tokens create deploy --app science-swap`.
