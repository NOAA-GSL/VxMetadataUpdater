# Docker Guide

This document explains how to build and run VxMetadataUpdater with Docker.

## Prerequisites

- Docker Desktop (or Docker Engine + Docker Compose plugin)
- A Couchbase credentials file at `$HOME/credentials`

## Files Used

- Docker image build config: `Dockerfile`
- Container runtime config: `docker-compose.yml`
- App settings mounted into the container: `settings.json`

## Build The Container Image

From the repository root, run:

    docker compose build

This builds the `vxmetadataupdater:local` image defined in `docker-compose.yml`.

## Run The Container

Run with default settings:

    docker compose up

Run and rebuild in one command:

    docker compose up --build

The container runs the app entrypoint and passes:

- `-c /run/config/credentials`
- `-s /app/settings.json`

Compose bind-mounts credentials from `$HOME/credentials` to `/run/config/credentials`.

The Compose environment sets:

- `BUCKET_READY_TIMEOUT_SECONDS=60` (override when slower clusters need more time)

## Optional Runtime Flags

The Compose file supports these optional environment variables:

- `VX_APP`: mapped to `-a` (run a single app)
- `VX_OUTPUT_PATH`: mapped to `-p` (write metadata output to a file)

Examples:

Run only one app:

    VX_APP=ceiling docker compose up --build

Write output to a mounted file under `./output`:

    VX_OUTPUT_PATH=/app/output/metadata.json docker compose up --build

If you use `-p`, also pass `-a` so only one metadata document is selected for output.

## Docker Bind Mount Setup

The current Compose configuration reads bind-mounted files from your host:

- credentials file from `$HOME/credentials`

Make sure the credentials file exists before running `docker compose up`.

## Stopping The Container

To stop:

    docker compose down

## Troubleshooting

- If Docker says a bind mount source file is missing, verify `$HOME/credentials` exists.
- If app startup fails due to credentials permissions, run `chmod 600 ~/credentials`.
- If startup fails with bucket readiness timeouts, increase `BUCKET_READY_TIMEOUT_SECONDS`.
- If Compose command is not found, install Docker Desktop or the Docker Compose plugin.
