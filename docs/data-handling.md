# Data handling

Use synthetic test data for recordings. Kipuray stores cases locally by default;
that does not automatically make captured page text or payloads safe to share.

## Capture

Input values are masked in visual recordings. Executable input actions are
redacted unless the element explicitly uses `data-kipuray-public`. Password,
hidden, and file inputs are not recorded as value actions. Cookies and authorization
headers are not exported, and a dedicated profile avoids inheriting your normal
browser session.

JSON API body capture is opt-in. Recognized credential keys such as passwords,
tokens, authorization, cookies, and API keys are replaced recursively. Sensitive
query keys are redacted. Affected fixtures are marked for review. These rules are
heuristics: they cannot recognize arbitrary personal information or secrets
embedded in ordinary text. Review DOM text, URLs, responses, and diagnostics.

Exclude a subtree with `data-private` or `data-kipuray-block`. Mask its text with
`data-kipuray-mask`. Application source scripts are excluded from DOM capture.

The workspace stores a checkpoint every three seconds during capture. Incomplete
checkpoints are inspectable after a crash. Delete unwanted cases from the workspace.

## Sharing and import

Export creates a portable file containing the reviewed case. It is not encrypted.
Sharing creates a bearer link valid for 24 hours. Anyone with the link can read the
recorded data until the link expires or the owner revokes it. Share-token hashes
are stored in SQLite; the raw token is returned once when creating the link.

Imported files are validated and displayed as data. Opening a case does not run
it. Execution requires a local target selected by the user and does not execute
shell commands from a file.

## Reports

Execution evidence can include browser screenshots and diagnostic URLs. Treat CI
artifacts and exported JSON/HTML reports with the same care as the original case.
Kipuray has no cloud account requirement and sends no product analytics.
