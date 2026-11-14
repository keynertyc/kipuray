# Changelog

## 0.1.1

- Wait for complete browser requests before evaluating functional expectations;
  check uncaught errors after the other expectations, regardless of their order.
  Slow fixtures now use the run deadline instead of a separate one-second drain.
- Preserve integer precision in recorded JSON and request matching. Distinct
  large numeric IDs no longer collapse into the same fixture key. Re-record cases
  captured with 0.1.0 if their payloads contained affected large integers.
- Prevent `run` and `compare` reports from overwriting their input `.kpr` file,
  including through symbolic links and hard links.
- Honor `doctor --chrome PATH` and `doctor --help`. Check standard Windows and
  per-user macOS browser installations, and use the discovered browser consistently.
- Fix HTML report rendering with CRLF templates in Windows source builds.
- Add regression coverage and run system tests on macOS and Windows as well as
  the existing Linux browser/UI checks and six-platform compilation.

## 0.1.0

- Initial release: portable browser recordings, JSON response variants,
  before/after fix verification, CLI reports, and an authenticated sharing server.
