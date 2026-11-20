FROM node:24-bookworm-slim AS frontend
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY migrations/ migrations/
COPY --from=frontend /src/web/dist/ web/dist/
RUN CGO_ENABLED=0 go build -trimpath -o /science-swap . \
    && install -d -m 1777 /runtime/tmp

FROM scratch
COPY --from=backend /runtime/ /
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend /science-swap /science-swap
USER 65532:65532
EXPOSE 8080
CMD ["/science-swap"]
