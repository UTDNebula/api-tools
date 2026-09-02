# How to Contribute

Thank you for your interest in contributing to `api-tools`! This guide covers our development workflow, coding standards, and more

## 1. Find something to work on

<!-- Don't worry, you don't have to be perfect, we appreciate anything you can do! -->
<!-- Look at current issues, or make a new one -->
<!-- Ask to be assigned the issue, dicuss the issue on GitHub, Discord, or at project meetings -->



## 2. Create a branch

Nebula recruits and members should make their changes on a branch, external contributors should work off of a fork

Our branch naming convention is `<issue-number>-<short-description-of-issue>` for example, `738-getting-started-docs`

## 3. Make Your Changes

Don't forget to format and check your code. We have helper scripts:

```bash
make check
```
or
```cmd
.\build.bat check
```

This executes several commands to check and format your code

Run the test suite to ensure no regressions were introduced:

- **Linux / macOS**:
  ```bash
  make test
  ```
- **Windows**:
  ```cmd
  build.bat test
  ```


## 4. Submitting a Pull Request (PR)

1. **Commit Your Work**: Write concise, descriptive commit messages:
   ```bash
   git commit -m "feat(scrapers): add degree scraper for catalog 2026"
   ```
2. **Push to GitHub**:
   ```bash
   git push origin feat/your-feature-name
   ```
3. **Open a PR**:
   - Navigate to the repository on GitHub and click **Compare & pull request**.
   - Provide a clear summary of what changes were made and why.
   - Mention any related issue numbers (e.g., `Closes #12`).
   - Confirm that `make check` and `make test` pass cleanly.
4. **Address Review Feedback**: Maintainers will review your PR and provide suggestions or approval.

---

## Next Step

See [Guides.md](Guides.md)
