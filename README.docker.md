# Docker Guide

This document explains how to build and run VxMetadataUpdater with Docker.

## Prerequisites

- Docker Desktop or Docker Engine
- Docker Compose plugin, only if you use the Compose commands
- A Couchbase credentials file at `$HOME/credentials`
- Your host UID/GID values (used to map secret ownership in Compose)

## Files Used

- Docker image build config: `Dockerfile`
- Optional Compose runtime config: `docker-compose.yml`
- Default app settings baked into image at `/app/settings.json` (from repo `settings.json`)

The Compose file is optional. You can build and run the same image directly with
Docker commands.

## Build The Container Image

From the repository root, run:

    docker compose build

This builds the `vxmetadataupdater:local` image defined in `docker-compose.yml`.

Without Compose, run:

    docker build -t vxmetadataupdater:local .

This builds the same local image from `Dockerfile`.

## Run The Container Without Compose

These examples use the published `ghcr.io/noaa-gsl/vxmetadataupdater:latest`
image and pass `--pull always` so Docker checks for a newer image before each
run.

From the repository root, make sure the host output directory exists and the
credentials file has secure permissions:

    mkdir -p output
    chmod 600 "$HOME/credentials"

Run with default settings:

```bash
docker run --rm \
  --pull always \
  --user "$(id -u):$(id -g)" \
  -v "$HOME/credentials:/run/config/credentials:ro" \
  -v "$PWD/output:/app/output" \
  -e BUCKET_READY_TIMEOUT_SECONDS=60 \
  ghcr.io/noaa-gsl/vxmetadataupdater:latest \
  -c /run/config/credentials \
  -s /app/settings.json
```

The `--user` option lets the container read a `0600` credentials file owned by
your host user. The app rejects credentials files that are readable by group or
others.

Run only one app:

```bash
docker run --rm \
  --pull always \
  --user "$(id -u):$(id -g)" \
  -v "$HOME/credentials:/run/config/credentials:ro" \
  -v "$PWD/output:/app/output" \
  -e BUCKET_READY_TIMEOUT_SECONDS=60 \
  ghcr.io/noaa-gsl/vxmetadataupdater:latest \
  -c /run/config/credentials \
  -s /app/settings.json \
  -a ceiling
```

Write output to a mounted file under `./output`:

```bash
docker run --rm \
  --pull always \
  --user "$(id -u):$(id -g)" \
  -v "$HOME/credentials:/run/config/credentials:ro" \
  -v "$PWD/output:/app/output" \
  -e BUCKET_READY_TIMEOUT_SECONDS=60 \
  ghcr.io/noaa-gsl/vxmetadataupdater:latest \
  -c /run/config/credentials \
  -s /app/settings.json \
  -a ceiling \
  -p /app/output/metadata.json
```

If you use `-p`, also pass `-a` so only one metadata document is selected for
output.

## Run The Container With Compose

Run with default settings:

    docker compose up

Run and rebuild in one command:

    docker compose up --build

The container runs the app entrypoint and passes:

- `-c /run/secrets/CREDENTIALS_FILE`
- `-s /app/settings.json`

No host settings bind mount is required. The image includes `/app/settings.json`.

Compose mounts credentials as a Docker secret named `CREDENTIALS_FILE`.
Set `VX_UID` and `VX_GID` so secret ownership matches the container runtime user:

    export VX_UID="$(id -u)"
    export VX_GID="$(id -g)"

Then run:

    docker compose up --build

The Compose environment sets:

- `BUCKET_READY_TIMEOUT_SECONDS=60` (override when slower clusters need more time)

## Optional Compose Runtime Flags

The Compose file supports these optional environment variables:

- `VX_APP`: mapped to `-a` (run a single app)
- `VX_OUTPUT_PATH`: mapped to `-p` (write metadata output to a file)

Examples:

Run only one app:

    VX_APP=ceiling docker compose up --build

Write output to a mounted file under `./output`:

    VX_OUTPUT_PATH=/app/output/metadata.json docker compose up --build

If you use `-p`, also pass `-a` so only one metadata document is selected for output.

## Docker Secret Setup

Compose reads credentials as a Docker secret sourced from:

- credentials file from `$HOME/credentials`

Make sure the credentials file exists before running the container.

## Optional Settings Override

If you want custom settings at runtime, bind mount another settings file over
`/app/settings.json`:

```bash
docker run --rm \
    --pull always \
    --user "$(id -u):$(id -g)" \
    -v "$PWD/my-settings.json:/app/settings.json:ro" \
    -v "$HOME/credentials:/run/config/credentials:ro" \
    -v "$PWD/output:/app/output" \
    ghcr.io/noaa-gsl/vxmetadataupdater:latest \
    -c /run/config/credentials \
    -s /app/settings.json
```

## Stopping The Container

If you used `docker run --rm` in the foreground, press `Ctrl+C` to stop it. The
container is removed automatically after it exits.

If you used Compose, run:

    docker compose down

## Troubleshooting

- If Docker says a bind mount source file is missing, verify `$HOME/credentials` exists.
- If app startup fails with `permission denied` on `/run/secrets/CREDENTIALS_FILE`, ensure `VX_UID` and `VX_GID` match the process user in the container.
- If app startup fails due to credentials permissions for direct bind mounts, run `chmod 600 ~/credentials`.
- If startup fails with bucket readiness timeouts, increase `BUCKET_READY_TIMEOUT_SECONDS`.
- If Compose command is not found, install Docker Desktop or the Docker Compose plugin.
