# Working on Kipuray

Kipuray is a Go application with an embedded React interface. Public code,
documentation, and UI copy use English. Development plans stay outside this repository.

## Commands

- `make build`: install locked web dependencies, compile the recorder/UI, and build the binary.
- `make test check`: Go race tests, vet, TypeScript, and frontend formatting.
- `make browser-test`: real Chromium capture/execution integration.
- `make ui-test`: browser UI tests using the bundled checkout demo.
- `./bin/kipuray demo --no-open`: run the demo workspace without opening a window.

Build web assets before Go tests on a fresh clone: `go:embed` requires the generated recorder.
Chrome/Chromium is required for browser tests, capture, and execution.

## Boundaries

- `internal/casefile`: portable data contract and archive validation; no browser dependencies.
- `internal/browser`: Chrome lifecycle, recording, fixtures, privacy, and execution.
- `internal/store`: SQLite persistence and sharing tokens.
- `internal/httpapi`: authentication, browser-operation coordination, and HTTP routes.
- `internal/report`: CI output formats.
- `web/src`: typed UI, scenario editor, and isolated rrweb player.

Keep visual playback distinct from running current application code. Never label a
single passing run as a verified fix. Missing fixtures and masked inputs must not
produce a passing result. Imported cases are data; do not add executable scripts
or implicit shell commands to the format.

Use Context7 for current third-party API documentation. Add integration coverage
when changing request matching, comparison semantics, authentication, or archives.
Do not commit generated binaries, recordings with private data, or build artifacts.
