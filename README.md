# API Tools

[![Commitizen friendly](https://img.shields.io/badge/commitizen-friendly-brightgreen.svg)](http://commitizen.github.io/cz-cli/)

_A CLI to scrape some really useful UTD data, parse it, and upload it to the Nebula API database for community use._

Part of [Project Nebula](https://about.utdnebula.com)

## Contributing

### Prerequisites

- Golang 1.18.4 (or higher)

### Development

Documentation for the project will be created soon, but for more information please visit our [Discord](https://discord.com/invite/tcpcnfxmeQ)

Clone the repository.

- The `grade-data` directory contains .CSV files of, you guessed it, the UTD grade data!
- The `main` directory contains the main file that runs the CLI.
- The `parser` directory contains the files and methods that parse the scraped data.
- The `scrapers` directory contains the scrapers for various UTD data sources.
- The `uploader` directory contains the uploader that sends the parsed data to the Nebula API database. (Under construction)

#### API Tools (Under construction)

The API Tools use Golang with Gin and the MongoDB Golang Driver.

### Deployment

[TBD]

## Questions or Feedback

If you have any questions about this project, reach out to the Project Nebula
maintainers at core-maintainers@utdnebula.com or open an issue or discussion on
this repository.
