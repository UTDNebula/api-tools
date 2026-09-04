# Project Structure & Codebase Map

This document provides an overview of the `api-tools` directory layout, explaining what each folder contains and guiding you on where to make changes when adding features or fixing bugs.

## Directory Overview

```text

api-tools/
├── data/                       # Files from scrapers and parsers
├── docs/                       # Source of wiki pages
├── logs/                       # Timestamped log files from runtime executions
├── parser/                     # Parsing, data normalization, and schema validation
│   ├── parser.go              # Central parsing orchestrator
│   ├── <domain>Parser.go      # Domain-specific parsers (courses, sections, maps, ASTRA, etc.)
│   ├── gradeLoader.go         # Loads historical grade distributions
│   ├── requisiteParser.go     # Parses prerequisite/corequisite strings
│   ├── validator.go           # Validates parsed structs before upload
│   └── testdata/              # HTML fixtures and expected JSON for regression tests
├── runners/                    # Shell automation scripts run on Google Cloud
├── scrapers/                   # Web scrapers for UTD websites, portals, and APIs
├── static-data/                # Static data that isn't scraped
│   ├── grades/                # Historical grade-distribution CSVs by semester
│   └── budgets/               # Historical budget PDFs
├── uploader/                   # Database ingestion modules for Nebula API / MongoDB
│   ├── database.go            # Establishes MongoDB connections
│   ├── uploader.go            # Coursebook uploads and merge/replace strategies
│   └── <domain>Uploader.go    # Domain-specific uploaders
├── utils/                      # Shared utilities
│   ├── methods.go             # ChromeDP, environment variables, headless mode, tokens
│   ├── logger.go              # Runtime logging to stdout and timestamped files
│   └── regexes.go             # Shared parsing and extraction regexes
├── .env.template               # Template for required environment variables
├── Dockerfile                  # Multi-stage container definition
├── Makefile                    # Build, check, and test tasks for Linux/macOS
├── build.bat                   # Build, check, and test tasks for Windows
├── cloudbuild.yaml             # Google Cloud Build deployment configuration
├── go.mod / go.sum             # Go dependency manifests
└── main.go                     # Application entrypoint and CLI flag definitions
```

## Next Step

See [How-to-Contribute.md](How-to-Contribute.md)
