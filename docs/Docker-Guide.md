# Docker & Containerization Guide

This guide explains how Docker is used in `api-tools`, how our multi-stage Docker build works, and how you can run automated tasks inside a local Docker container.

---

## Why Do We Use Docker?

Running web scrapers requires several system-level dependencies:

- **Chromium / Google Chrome**: Needed by ChromeDP for headless browser automation.
- **Poppler Utilities (`pdftotext`)**: Needed for converting academic calendar PDFs into raw text.
- **Go Runtime**: Needed to compile the application binary.
- **Google Cloud SDK (`gcloud`)**: Needed in production to access cloud secrets.

Instead of requiring every contributor to manually install and configure these system packages, Docker packages everything into a standardized, lightweight Linux container.
This ensures that scripts execute identically on developer laptops (Linux, macOS, Windows) and in production (Google Cloud Platform).

> [!NOTE]
> **New to Docker?** Check out the official [Docker 101 Tutorial](https://www.docker.com/101-tutorial/) or the [Docker Get Started Guide](https://docs.docker.com/get-started/) to learn the basics of images and containers.

---

## Understanding the Multi-Stage Dockerfile

Our [`Dockerfile`](file:///var/home/justin/Documents/Projects/api-tools/Dockerfile) uses Docker's multi-stage build pattern to keep production images secure, fast, and minimal.

1. **`builder` Stage (`golang:1.26`)**:
   - Copies Go source code.
   - Runs `make setup`, `make check` (vetting and linting), and `make build`.
   - Produces the compiled `api-tools` binary.

2. **`base` Stage (`debian:12-slim`)**:
   - A lightweight Debian base image.
   - Installs runtime dependencies: `chromium`, `poppler-utils`, and `google-cloud-sdk`.
   - Copies the compiled `api-tools` binary from the `builder` stage.
   - Copies `runners/` and `static-data/`.
   - Sets the entrypoint to [`runners/setup.sh`](file:///var/home/justin/Documents/Projects/api-tools/runners/setup.sh).

3. **`local` Stage (`FROM base AS local`)**:
   - Inherits from `base` and copies your local `.env` file into the container.
   - Ideal for local testing and debugging.

---

## Running Docker Locally

### Step 1: Ensure Your `.env` File Exists

Make sure you have copied `.env.template` to `.env` in the project root:

```bash
cp .env.template .env
```

### Step 2: Build the Local Docker Image

Build using the `--target local` flag to include your local `.env` file:

```bash
docker build --target local -t my-runner:local .
```

### Step 3: Run a Runner Script

Run the container by specifying the environment mode and target script name:

```bash
docker run --rm -e ENVIRONMENT=local -e RUNNER_SCRIPT_NAME=daily.sh my-runner:local
```

You can replace `daily.sh` with any runner in the `runners/` directory:

- `daily.sh` — Scrapes, parses, and uploads events (Astra, Mazevo, Comet Calendar).
- `weekly.sh` — Scrapes, parses, and uploads academic calendars, discounts, and degrees.
- `monthly.sh` — Scrapes, parses, and uploads map locations and budgets.

---

## Production vs. Local Execution (`runners/setup.sh`)

When the Docker container starts, it executes [`runners/setup.sh`](file:///var/home/justin/Documents/Projects/api-tools/runners/setup.sh):

- **Local Mode (`ENVIRONMENT=local`)**:
  - Skips cloud authentication.
  - Directly uses the `.env` file baked into the local image.
  - Runs `/app/runners/$RUNNER_SCRIPT_NAME`.

- **Production Mode (`ENVIRONMENT=gcp`)**:
  - Authenticates with Google Cloud Secret Manager using service account keys.
  - Dynamically fetches secrets and creates the `.env` file at runtime.
  - Runs `/app/runners/$RUNNER_SCRIPT_NAME`.

- Resolve any container or browser errors in the [**Troubleshooting Guide**](/docs/Troubleshooting.md).
