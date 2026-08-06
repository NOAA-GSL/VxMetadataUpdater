# Docker Guide

This document explains how to build and run VxMetadataUpdater with Docker.

## Prerequisites

- Docker Desktop (or Docker Engine + Docker Compose plugin)
- A Couchbase credentials file at `$HOME/credentials`
- A Capella root certificate at `$HOME/capella-root-certificate.pem` (required by the current Compose configuration)

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

- `-c /tmp/credentials`
- `-s /app/settings.json`

At startup, Compose mounts Docker secrets under `/run/secrets`, then the launch command copies credentials to `/tmp/credentials` and applies `chmod 600` so the app's file permission check passes.

## Optional Runtime Flags

The Compose file supports these optional environment variables:

- `VX_APP`: mapped to `-a` (run a single app)
- `VX_OUTPUT_PATH`: mapped to `-p` (write metadata output to a file)

Examples:

Run only one app:

    VX_APP=ceiling docker compose up --build

Write output to a mounted file under `./output`:

    VX_OUTPUT_PATH=/app/output/metadata.json docker compose up --build

## Docker Secrets Setup

The current Compose configuration reads secrets from host files:

- `credentials` secret from `$HOME/credentials`
- `capella_root` secret from `$HOME/capella-root-certificate.pem`

Make sure both files exist before running `docker compose up`.

## Stopping The Container

To stop:

    docker compose down

## Troubleshooting

- If Docker says a secret source file is missing, verify `$HOME/credentials` and `$HOME/capella-root-certificate.pem` exist.
- If app startup fails due to credentials permissions, ensure the startup command still includes:
  - copy from `/run/secrets/credentials` to `/tmp/credentials`
  - `chmod 600 /tmp/credentials`
- If Compose command is not found, install Docker Desktop or the Docker Compose plugin.
