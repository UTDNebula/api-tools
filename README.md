# API Tools

_A CLI to scrape some really useful UTD data, parse it, and upload it to the [Nebula API](https://github.com/utdnebula/nebula-api) for community use._

Project maintained by [Nebula Labs](https://about.utdnebula.com).

### Design

#### - The `scrapers` directory contains the scrapers for various UTD data sources. This is where the data pipeline begins.
  - The scrapers are concerned solely with data collection, not necessarily validation or processing of said data. Those responsibilities are left to the parsing stage.
#### - The `parser` directory contains the files and methods that parse the scraped data. This is the 'middle man' of the data pipeline.
  - The parsing stage is responsible for 'making sense' of the scraped data; this consists of reading, validating, and merging/intermixing of various data sources.
  - The input data is considered **immutable** by the parsing stage. This means the parsers should never modify the data being fed into them.
#### - The `uploader` directory contains the uploader that sends the parsed data to the Nebula API MongoDB database. This is the final stage of the data pipeline.
  - The uploader(s) are concerned solely with pushing parsed data to the database. Data, at this point, is assumed to be valid and ready for use.
#### - The `static-data/grades` directory contains .csv files of UTD grade data. 
  - Files are named by year and semester, with a suffix of `S`, `U`, or `F` denoting Spring, Summer, and Fall semesters, respectively.
  - This means that, for example, `22F.csv` corresponds to the 2022 Fall semester, whereas `18U.csv` corresponds with the 2018 Summer semester.
  - This grade data is collected independently from the scrapers, and is used during the parsing process.
#### - The `static-data/budgets` directory contains .pdf files of UTD budget data. 
  - Files are named by fiscal year.
  - This budget data is used as a backup of scraped data as some years have been removed from the website.

### Contributing

Please visit our [Discord](https://discord.utdnebula.com) and talk to us if you'd like to contribute!

### Prerequisites

- Golang 1.24 (or higher)

### Build

To build the project, simply clone the repository and then either:
  - Run `make` in the root (top level) directory (for systems with `make` installed, i.e. most Linux distros, MacOS)
  - Run `build.bat` on Windows systems (unless you want to deal with getting `make` to work on Windows :P)

The build process will output an executable file named `api-tools`; this executable is the CLI and can be ran in your terminal!

Additionally, you can run build (on Windows) and make (on MacOS/Linux) with the following arguments:
  - `setup`: Installs required dependencies for the tools.
  - `check`: Verifies prequisites and ensures the executable can be built.
  - `test`: Test run to see if the executable works after building
  - `build`: Builds the executble and makes it ready for use.

### Usage

The `api-tools` command line interface supports three main modes: scraping, parsing and uploading data to the Nebula API.

#### Environment Variables
Before being able to use the tool, configure the `.env` file by following these steps:
  1. Find the `.env.template` file and rename it to `.env`
  2. Specify the required credentials for your use case as a string ("") following the variable name.
    Example: LOGIN_NETID="ABC123456"

#### Basic Usage

Run the tool by changing directory using `cd` to the `api-tools` directory and running the executable with the appropriate flags in the command line. To see all available options with the tool, run: `./api-tools`. To enable logging for debugging, use the verbose flag: `./api-tools -verbose`. Find available flags for each mode in the following tables.

---

### Scraping Mode

| Command | Description |
|---------|-------------|
| `./api-tools -scrape -academicCalendars` | Scrapes academic calendar PDFs. |
| `./api-tools -scrape -astra` | Scrapes Astra data. |
| `./api-tools -scrape -cometCalendar` | Scrapes Comet Calendar data. |
| `./api-tools -scrape -coursebook -term 24F` | Scrapes coursebook data for Fall 2024.<br>• Use `-resume` to continue from last prefix.<br>• Use `-startprefix [prefix]` to begin at a specific course prefix. |
| `./api-tools -scrape -discounts` | Scrapes discount programs. |
| `./api-tools -scrape -degrees` | Scrapes degrees data |
| `./api-tools -scrape -map` | Scrapes UTD Map data. |
| `./api-tools -scrape -mazevo` | Scrapes Mazevo data. |
| `./api-tools -scrape -profiles` | Scrapes UTD professor profiles. |
| `./api-tools -scrape -headless` | Runs ChromeDP in headless mode. |
| `./api-tools -o [directory]` | Sets output directory (default: `./data`). |

For profile scraping, you can optionally scope requests by school to reduce API load:
- Set `PROFILE_SCHOOLS` to a comma/semicolon/space-separated list (example: `PROFILE_SCHOOLS=ECS;BBS;AHT`).
- Then run `./api-tools -scrape -profiles` as usual.
- If `PROFILE_SCHOOLS` is not set, the scraper defaults to batched `person` slug requests.

### Parsing Mode:

| Command | Description |
|---------|-------------|
| `./api-tools -parse -academicCalendars` | Parses academic calendar PDFs. |
| `./api-tools -parse -astra` | Parses Astra data. |
| `./api-tools -parse -cometCalendar` | Parses Comet Calendar data. |
| `./api-tools -parse -csv [directory]` | Outputs grade data CSVs (default: `./static-data/grades`). |
| `./api-tools -parse -discounts` | Parses discount programs HTML. |
| `./api-tools -parse -degrees` | Parses degrees from HTML. |
| `./api-tools -parse -map` | Parses UTD Map data. |
| `./api-tools -parse -mazevo` | Parses Mazevo data. |
| `./api-tools -parse -skipv` | Skips post-parse validation (**use with caution**). |


### Upload Mode:
| Command | Description |
|---------|-------------|
| `./api-tools -upload -academicCalendars` | Uploads academic calendars. |
| `./api-tools -upload -discounts` | Uploads discount programs. |
| `./api-tools -upload -events` | Uploads Astra, Mazevo, and Comet Calendar data. |
| `./api-tools -upload -map` | Uploads UTD Map data. |
| `./api-tools -upload -replace` | Replaces old data instead of merging. |
| `./api-tools -upload -static` | Uploads only static aggregations. |

Additionally, you can use the `-i [directory]` flag to specify where to read data from (default: `./data`) and the `-l [directory]` flag to specify where logs must be dumped (default: `./logs`).

#### Docker

Docker is used for automated running on Google Cloud Platform. More info [here](https://nebula-labs.atlassian.net/wiki/x/AYBjFw).

To build the container for local testing first make sure all scripts in the `runners` folder have LF line endings then run:
```
docker build --target local -t my-runner:local .
docker run --rm -e ENVIRONMENT=local -e RUNNER_SCRIPT_NAME=daily.sh my-runner:local
```

## Questions?
Reach out to the team on [Discord](https://discord.utdnebula.com) and with any questions you may have!
