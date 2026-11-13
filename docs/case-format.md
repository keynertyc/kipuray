# Portable case format

A `.kpr` file is a ZIP archive containing exactly `manifest.json` and `case.json`.
The manifest holds format version `1` and the SHA-256 digest of the uncompressed
case document. Hashes detect corruption; they do not authenticate a case's author.

The importer rejects unexpected/duplicate entries, unsupported versions, invalid
IDs, invalid action/expectation types, invalid viewport/status ranges, and archives
larger than 64 MiB compressed or expanded. It never extracts arbitrary paths.
Imported cases receive a new workspace identity.

The case document contains:

| Field | Content |
|---|---|
| `id`, `version`, `createdAt`, `title` | Case identity and metadata |
| `origin`, `width`, `height` | Original URL and viewport |
| `events` | rrweb snapshots and mutations for visual playback |
| `actions` | `click`, `input`, `select`, or `navigate` steps |
| `fixtures` | Method, URL, request body, status, JSON body, content type, delay |
| `expectations` | `text`, `visible`, `url`, or `no-errors` checks |
| `variants` | One replacement fixture per named variant |
| `diagnostics`, `warnings` | Evidence and explicit recording limitations |
| `executionProblems` | Unsupported or ambiguous capture conditions |
| `recordingState`, `truncated` | Whether a capture completed within its limits |

Input actions and fixtures can carry a `redacted` flag. A redacted value must be
replaced with synthetic data and reviewed before execution. At least one functional
expectation is required; `no-errors` alone does not qualify.

The frontend origin can be explicitly remapped to a local target. Other API origins
retain their original identity and are still fulfilled through fixtures. Request
matching canonicalizes query ordering and valid JSON bodies; repeated matches are
consumed in order. An unused or unexpected request makes the scenario incomplete.

There are no application binaries, cookies, browser profiles, shell scripts, or
databases in the container. Visual playback can work without the original app;
execution needs a running local application and Chromium.
