# How to Contribute

Thank you for your interest in contributing to `api-tools`! This guide covers our development workflow, coding standards, and more.
Don't worry if you don't quite know what you're doing, we're here to help. We don't expect perfection and appreciate anything you can do to help!

## Find something to work on

Look at our current issues, or make your own on our [issues page](../issues).

Once you find an issue, write a comment asking if you can work on a particular issue.

Be sure to reference [[getting-started-with-api-tools|Getting Started.md]] to create the project.

## Create a branch

Nebula recruits and members should make their changes on a branch, external contributors should work off of a fork as they do not have permission to make a branch.

Our branch naming convention is `<issue-number>-<short-description-of-issue>` for example, `738-new-developer-docs`.

## Make Your Changes

It's time to code!

Don't forget to format and check your code. We have helper scripts with **Make** and **build.bat**:

```bash
make check
```

or

```cmd
.\build.bat check
```

Run tests with

```bash
make test
```

or

```cmd
.\build.bat test
```

We would appreciate if you make a draft Pull Request as your working on it, so we can see your progress and help you out!

## Making a Pull Request (PR)

Open [our Pull Request Page](../pulls), and create a PR. If you're not finished, you can make your PR a draft.

Maintainers will review your PR and provide suggestions or approval.

## Next Step

See [Guides.md](Guides.md)
