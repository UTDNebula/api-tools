# Troubleshooting

This guide gives solutions to common problems developers or users of this project may run into.

## Enabling Verbose Debug Logging

If a scraper or parser is behaving unexpectedly, re-run the command with the `-verbose` flag:

For example:

```bash
./api-tools -verbose -scrape -coursebook -term 24F
```

This will give more logs, and can help you better understand what's going on.

## "exec: 'chromium': executable file not found in $PATH" or Chrome fails to launch

ChromeDP cannot find a Chromium or Google Chrome binary on your operating system. You'll need to install it.

## KEY is missing from .env

Ensure you have filled environment variables in .env. If you need help, ask for help!

## Still Stuck?

Ask for help on [Discord](https://discord.utdnebula.com)!
