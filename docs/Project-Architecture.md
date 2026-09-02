# Pipeline Architecture & Core Concepts

`api-tools` is built around a unidirectional, staged data pipeline. Each stage has a single, well-defined responsibility. Understanding this separation of concerns is essential before modifying or adding code.

---

## High-Level Data Flow

The lifecycle of data moving through `api-tools` follows a four-stage progression:

```mermaid
flowchart LR
    A[UTD Data Sources\nWebsites, APIs, PDFs] -->|1. Scraper| B[(Raw Data Files\ndata/)]
    B -->|2. Parser| C{Validator\nvalidator.go}
    D[(Static Data\nstatic-data/)] -->|2. Parser| C
    C -->|Validated Models| E[3. Uploader]
    E -->|4. MongoDB Insert/Merge| F[(Nebula API Database)]
    
    subgraph Automation
    G[Runners\nrunners/*.sh] -.->|Orchestrates| A
    G -.->|Orchestrates| B
    G -.->|Orchestrates| E
    end
```

---

## The Four Core Components

### 1. Scrapers (`scrapers/`) — *Data Acquisition*
- **What they do**: Connect to external UTD web servers, APIs, and portals to download raw information.
- **How they work**: They use HTTP requests (via standard Go `net/http`) or headless browser automation with [ChromeDP](https://github.com/chromedp/chromedp) to navigate pages, fill forms, trigger clicks, and capture raw HTML, JSON, or PDF files.
- **Rule of Thumb**: **Scrapers do NOT validate or parse data.** Their sole job is to faithfully capture raw data from the source and save it to disk in the `data/` directory (e.g., `data/24f/cp_cs/cs1337.001.24f.html`).
- **External Resources**:
  - [MDN Web Docs: Introduction to the DOM](https://developer.mozilla.org/en-US/docs/Web/API/Document_Object_Model/Introduction)
  - [ChromeDP Examples](https://github.com/chromedp/examples)

---

### 2. Parsers (`parser/`) — *Data Transformation & Validation*
- **What they do**: Read raw files produced by scrapers alongside static reference files, extract meaningful fields, normalize inconsistent formatting, and assemble validated Go data structures.
- **How they work**:
  - They parse HTML using Go HTML tokenizers and CSS query selectors (via `golang.org/x/net/html`).
  - They cross-reference scraped records with static datasets (such as grade distribution CSVs in `static-data/grades` or budget PDFs in `static-data/budgets`).
  - They run the parsed structs through validation checks in [`parser/validator.go`](file:///var/home/justin/Documents/Projects/api-tools/parser/validator.go) to ensure all required fields conform to the Nebula API schema.
- **Core Principle: Immutability**: Input files in `data/` are strictly **immutable**. Parsers must treat input files as read-only and never modify raw scraped data.
- **External Resources**:
  - [Go `html` package documentation](https://pkg.go.dev/golang.org/x/net/html)
  - [Go Regular Expressions (`regexp`) Guide](https://pkg.go.dev/regexp)

---

### 3. Uploaders (`uploader/`) — *Database Synchronization*
- **What they do**: Take validated data models and push them to the [Nebula API](https://github.com/utdnebula/nebula-api) database (MongoDB).
- **How they work**:
  - Connect to MongoDB using the official Go Mongo driver (`go.mongodb.org/mongo-driver`).
  - Support both **merge** operations (updating existing records with new fields) and **replace** operations (overwriting outdated datasets).
  - Compute static aggregation metrics where required before saving.
- **Rule of Thumb**: Uploaders assume data passed to them has already been parsed and validated.
- **External Resources**:
  - [MongoDB Go Driver Tutorial](https://www.mongodb.com/docs/drivers/go/current/)

---

### 4. Runners (`runners/`) — *Automated Orchestration*
- **What they do**: Shell scripts (`.sh`) that coordinate automated end-to-end execution of the pipeline on a scheduled cron cadence.
- **How they work**:
  - Instead of manually typing CLI commands, runners chain together scraping, parsing, and uploading steps for specific domains.
  - Examples in `runners/`:
    - `daily.sh`: Daily scrape/parse/upload for volatile event data (Astra, Mazevo, Comet Calendar).
    - `weekly.sh`: Weekly scrape/parse/upload for academic calendars, discount programs, and degree plans.
    - `monthly.sh`: Monthly updates for campus map locations and budget reports.
    - `setup.sh`: Docker entrypoint script that retrieves secrets from Google Cloud Secret Manager (in production) before launching a specified runner.
- **External Resources**:
  - [Bash Scripting Tutorial for Beginners](https://linuxconfig.org/bash-scripting-tutorial-for-beginners)

---

## Component Comparison

| Component | Directory | Input | Output | Mutation / Side Effects | Key Goal |
|---|---|---|---|---|---|
| **Scraper** | `scrapers/` | UTD Websites & APIs | Raw files in `data/` (`.html`, `.json`, `.pdf`) | Writes raw data to disk | Reliable network extraction |
| **Parser** | `parser/` | `data/` + `static-data/` | Validated Go data models | **None** (inputs are read-only) | Structure & schema validation |
| **Uploader** | `uploader/` | Validated Go data models | MongoDB Collections | Inserts / updates database records | Database persistence |
| **Runner** | `runners/` | Scheduled Cron / Trigger | Executes CLI binary sequence | Runs multiple stages in sequence | End-to-end pipeline automation |

---

## Why Separate These Concerns?

1. **Testability & Offline Development**: Because scrapers save raw data to `data/`, parsers can be tested and developed offline without repeatedly pinging university servers or needing internet access.
2. **Resilience**: If a university website layout changes, only the scraper or selector logic needs updating; the uploader and downstream database logic remain completely unaffected.
3. **Reproducibility**: If an issue occurs during parsing or uploading, the raw input data is preserved on disk, allowing developers to inspect exact HTML responses and write reproducible test fixtures.

---

## Next Step
See [Project-Structure.md](Project-Structure.md) 

