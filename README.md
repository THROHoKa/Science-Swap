# Science Swap

Science Swap is a planned web app that helps academics find a partner at another university for a semester exchange. The proposed matching workflow uses academic field, overlapping availability, and preferred region to suggest potential partners. An exchange match requires interest from both people.

## Project status

The project is in the planning stage. Application code, database migrations, and local run commands are still to come.

## Planned stack

- **Backend** — Go with `net/http`, PostgreSQL 17 or newer, `pgx` for database access, and `goose` for migrations.
- **Frontend** — Vue 3, TypeScript, Vite, and Tailwind CSS, with German and English interfaces.
- **Deployment** — A single Go binary with the frontend embedded, packaged in a Docker container for Fly.io in Frankfurt.
