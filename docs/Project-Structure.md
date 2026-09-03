# Project Structure & Codebase Map

This document provides an overview of the `api-tools` directory layout, explaining what each folder contains and guiding you on where to make changes when adding features or fixing bugs.

## Directory Overview

```text
api-tools/
├── data/                  # Files from scrapers and parsers
├── docs/                  # Source of wiki pages
├── logs/                  # Timestamped log files from runtime executions
├── parser/                # Parsing, data normalization, and schema validation
│   └── testdata/          # HTML input fixtures and expected JSON, used for tests
├── runners/               # Shell automation scripts ran on Google Cloud (daily, weekly, monthly)
├── scrapers/              # Web scrapers for UTD websites, portals, and APIs
├── static-data/           # Static data that isn't scraped
├── uploader/              # Database ingestion modules for Nebula API / MongoDB
├── utils/                 # Shared utilities (ChromeDP helpers, logger, regexes)
├── .env.template          # Template for required environment variables
├── Dockerfile             # Multi-stage container definition
├── Makefile               # Task runner for Linux and macOS (build, check, test)
├── build.bat              # Task runner for Windows (build, check, test)
├── cloudbuild.yaml        # Google Cloud Build deployment configuration
├── go.mod / go.sum        # Go dependency manifests
└── main.go                # Application entrypoint and CLI flag definitions
```

## Directory Breakdown

### `parser/`
Responsible for interpreting raw files, sanitizing fields, resolving cross-references, and ensuring schema compliance:
- `<domain>Parser.go`: Dedicated parser files corresponding to each scraper (e.g., `courseParser.go`, `sectionParser.go`, `mapParser.go`, `astraParser.go`).
- `parser.go`: The central orchestrator that coordinates parsing of scraped coursebook files with static grade and profile data.
- `gradeLoader.go`: Reads historical grade distribution CSVs from `static-data/grades`.
- `requisiteParser.go`: Parses prerequisite and corequisite strings into structured dependency trees.
- `validator.go`: Validates that parsed Go structs contain all non-empty mandatory fields before upload.
- `testdata/`: Test suites containing realistic sample input files (`input.html`) and expected JSON outputs (`course.json`, `section.json`, `professors.json`) for automated regression testing.

### `uploader/`
Handles pushing structured, validated data into MongoDB:
- `database.go`: Establishes MongoDB client connections using `MONGODB_URI`.
- `uploader.go`: Core coursebook data uploader with merge/replace strategies.
- `<domain>Uploader.go`: Specialized uploaders for events, maps, discounts, degrees, and academic calendars.

### `static-data/`
Persistent reference datasets that cannot be scraped on-demand or serve as historical baselines:
- `static-data/grades/`: CSV files organized by semester (e.g., `22F.csv` for Fall 2022, `23S.csv` for Spring 2023, `23U.csv` for Summer 2023).
- `static-data/budgets/`: Historical PDF budget files for fiscal years where web links are no longer available.

### `utils/`
Shared helper functions used across scrapers, parsers, and uploaders:
- `methods.go`: ChromeDP initialization, headless settings, environment variable readers (`GetEnv`), and token refresh helpers.
- `logger.go`: Dual logger implementation (`SplitWriter`) that writes output simultaneously to `stdout` and timestamped files in `logs/`.
- `regexes.go`: Compiled regular expressions for text parsing and course extraction.

### Root Configuration Files
- `main.go`: The entrypoint of the CLI application. Parses command line flags (like `-scrape`, `-parse`, `-upload`, `-verbose`, `-headless`) and routes execution to the appropriate package functions.
- `Makefile` / `build.bat`: Scripts to automate setup, formatting checks (`staticcheck`, `gofmt`, `goimports`, `go vet`), compiling the binary, and running unit tests.
- `Dockerfile`: Defines the multi-stage Docker build for local testing and production deployment.

## Next Step
See [How-to-Contribute.md](How-to-Contribute.md) 
