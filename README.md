# API Tools

_A CLI to scrape some really useful UTD data, parse it, and upload it to the [Nebula API](https://github.com/utdnebula/nebula-api) for community use._

Project maintained by [Nebula Labs](https://about.utdnebula.com).

### Design

#### - The `grade-data` directory contains .csv files of UTD grade data. 
  - Files are named by year and semester, with a suffix of `S`, `U`, or `F` denoting Spring, Summer, and Fall semesters, respectively.
  - This means that, for example, `22F.csv` corresponds to the 2022 Fall semester, whereas `18U.csv` corresponds with the 2018 Summer semester.
  - This grade data is collected independently from the scrapers, and is used during the parsing process.
#### - The `scrapers` directory contains the scrapers for various UTD data sources. This is where the data pipeline begins.
  - The scrapers are concerned solely with data collection, not necessarily validation or processing of said data. Those responsibilities are left to the parsing stage.
#### - The `parser` directory contains the files and methods that parse the scraped data. This is the 'middle man' of the data pipeline.
  - The parsing stage is responsible for 'making sense' of the scraped data; this consists of reading, validating, and merging/intermixing of various data sources.
  - The input data is considered **immutable** by the parsing stage. This means the parsers should never modify the data being fed into them.
#### - The `uploader` directory contains the uploader that sends the parsed data to the Nebula API MongoDB database. This is the final stage of the data pipeline.
  - The uploader(s) are concerned solely with pushing parsed data to the database. Data, at this point, is assumed to be valid and ready for use.
#### - The `generator` directory contains code to create data from scratch.
  - This is part of a seperate pipeline (generator > uploader instead of scraper > parser > uploader) for data that does not come from an external source.

### Contributing

Please visit our [Discord](https://discord.utdnebula.com) and talk to us if you'd like to contribute!

### Prerequisites

- Golang 1.23 (or higher)

### Development

Documentation for the project will be created soon, but for more information please visit our [Discord](https://discord.utdnebula.com).

To build the project, simply clone the repository and then either:
  - Run `make` in the root (top level) directory (for systems with `make` installed, i.e. most Linux distros, MacOS)
  - Run `build.bat` on Windows systems (unless you want to deal with getting `make` to work on Windows :P)

The build process will output an executable file named `api-tools`; this executable is the CLI and can be ran in your terminal!
