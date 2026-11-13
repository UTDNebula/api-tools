# API Tools

_A CLI toolchain to scrape, parse, validate, and upload UTD university data to the [Nebula API](https://github.com/utdnebula/nebula-api) database._

Project maintained by [Nebula Labs](https://about.utdnebula.com).

> **Developer Wiki**: If you are interested in developing this project, head over to the [**Developer Wiki**](docs/Home.md)!

---

## Quick Navigation

- [Developer Wiki](docs/Home.md)
- [Getting Started Guide](/docs/Getting-Started.md)
- [Pipeline Architecture (Scraper -> Parser -> Uploader)](/docs/Project-Architecture.md)
- [Project Structure & Codebase Map](/docs/Project-Structure.md)
- [Docker & Container Guide](/docs/Docker-Guide.md)
- [Contributing Guide](/docs/How-to-Contribute.md)
- [Troubleshooting & FAQ](/docs/Troubleshooting.md)
- [Discord Community](https://discord.utdnebula.com)

---

## Prerequisites & Building

- **Go 1.24+**
- **Make** (Linux/macOS)

To build the executable binary:

```bash
# Linux / macOS
make setup    # Install linter tools (staticcheck, goimports)
make check    # Verify formatting, vetting, and linting
make build    # Compiles ./api-tools binary
make test     # Run automated test suite

# Windows
build.bat setup
build.bat check
build.bat build   # Compiles api-tools.exe
build.bat test
```

---

## CLI Usage Reference

The `api-tools` command line interface supports three main modes: **scraping**, **parsing**, and **uploading**.

### Global Flags

| Flag         | Default  | Description                                                              |
| ------------ | -------- | ------------------------------------------------------------------------ |
| `-i <dir>`   | `./data` | Input directory to read data from                                        |
| `-o <dir>`   | `./data` | Output directory to write scraped/parsed data to                         |
| `-l <dir>`   | `./logs` | Directory to write timestamped log files to                              |
| `--verbose`  | `false`  | Enables verbose debug logging with file names and microsecond timestamps |
| `--headless` | `false`  | Runs ChromeDP browser automation in background headless mode             |

---

### Scraping Mode (`--scrape`)

Scrapes raw data from UTD sources and saves it to `./data` (or the directory specified by `-o`).

| Command                                        | Description                                                                                                                                                                        |
| ---------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `./api-tools --scrape --coursebook --term 24F` | Scrapes CourseBook data for Fall 2024.<br>• `--resume`: Resume from last completed course prefix.<br>• `--startprefix <prefix>`: Begin at a specific course prefix (e.g. `cp_cs`). |
| `./api-tools --scrape --evals --term 24F`      | Scrapes CourseBook course evaluation data for the specified term.                                                                                                                  |
| `./api-tools --scrape --profiles`              | Scrapes faculty and professor directory profiles.                                                                                                                                  |
| `./api-tools --scrape --astra`                 | Scrapes Astra schedule and room reservation data.                                                                                                                                  |
| `./api-tools --scrape --mazevo`                | Scrapes Mazevo event reservation data via API.                                                                                                                                     |
| `./api-tools --scrape --cometCalendar`         | Scrapes public UTD Comet Calendar events.                                                                                                                                          |
| `./api-tools --scrape --map`                   | Scrapes UTD campus map building and parking coordinates.                                                                                                                           |
| `./api-tools --scrape --degrees`               | Scrapes academic degrees and catalog majors.                                                                                                                                       |
| `./api-tools --scrape --discounts`             | Scrapes student discount program listings.                                                                                                                                         |
| `./api-tools --scrape --academicCalendars`     | Scrapes academic calendar PDF documents.                                                                                                                                           |
| `./api-tools --scrape --budgets`               | Scrapes university budget reports.                                                                                                                                                 |

---

### Parsing Mode (`-parse`)

Reads raw files from `./data` (or `-i`), normalizes records against `static-data/`, validates data models, and outputs structured schemas.

| Command                                 | Description                                                                                 |
| --------------------------------------- | ------------------------------------------------------------------------------------------- |
| `./api-tools -parse`                    | Runs full coursebook parsing pipeline across all scraped prefixes.                          |
| `./api-tools -parse -gradesDir <path>`  | Specifies custom path for grade distribution CSVs (default: `./static-data/grades`).        |
| `./api-tools -parse -astra`             | Parses scraped Astra scheduling data.                                                       |
| `./api-tools -parse -mazevo`            | Parses scraped Mazevo event data.                                                           |
| `./api-tools -parse -cometCalendar`     | Parses scraped Comet Calendar events.                                                       |
| `./api-tools -parse -map`               | Parses campus map data.                                                                     |
| `./api-tools -parse -degrees`           | Parses degree plans from HTML.                                                              |
| `./api-tools -parse -discounts`         | Parses student discount listings from HTML.                                                 |
| `./api-tools -parse -academicCalendars` | Parses academic calendar dates from PDFs.                                                   |
| `./api-tools -parse -budgets`           | Parses budget reports from PDFs (use `-useBackupBudgets` to include `static-data/budgets`). |
| `./api-tools -parse -skipv`             | Skips post-parse schema validation (**use with caution**).                                  |

---

### Upload Mode (`-upload`)

Pushes validated data to the MongoDB database specified in `MONGODB_URI`.

| Command                                  | Description                                                               |
| ---------------------------------------- | ------------------------------------------------------------------------- |
| `./api-tools -upload`                    | Uploads parsed coursebook data (merges into existing records by default). |
| `./api-tools -upload -replace`           | Overwrites/replaces existing records instead of merging.                  |
| `./api-tools -upload -static`            | Uploads static aggregations only.                                         |
| `./api-tools -upload -events`            | Uploads Astra, Mazevo, and Comet Calendar event datasets.                 |
| `./api-tools -upload -map`               | Uploads campus map locations.                                             |
| `./api-tools -upload -degrees`           | Uploads degree plans.                                                     |
| `./api-tools -upload -discounts`         | Uploads discount programs.                                                |
| `./api-tools -upload -academicCalendars` | Uploads academic calendar dates.                                          |
| `./api-tools -upload -budgets`           | Uploads budget data.                                                      |

---

## Docker & Automated Runners

Automated scheduled scraping is executed via Docker containers and cloud cron jobs:

```bash
# Build the container for local testing
docker build --target local -t my-runner:local .

# Run daily event tasks
docker run --rm -e ENVIRONMENT=local -e RUNNER_SCRIPT_NAME=daily.sh my-runner:local

# Run weekly catalog tasks
docker run --rm -e ENVIRONMENT=local -e RUNNER_SCRIPT_NAME=weekly.sh my-runner:local

# Run monthly map & budget tasks
docker run --rm -e ENVIRONMENT=local -e RUNNER_SCRIPT_NAME=monthly.sh my-runner:local
```

For more details on Docker execution and cloud configuration, see the [Docker Guide](/docs/Docker-Guide.md).

---

## Community & Support

Have questions, suggestions, or want to contribute? Reach out on our [Discord](https://discord.utdnebula.com)!
