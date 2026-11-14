# Test your own application with Kipuray

This walkthrough covers the released v0.1 workflow: install Kipuray, capture a bug
in your local application, and check the same case after changing your code.
You do not need to clone Kipuray or add its SDK to your application.

[Install](#1-install-the-right-binary) · [Start your app](#2-start-your-application-locally) ·
[Record](#3-record-the-bug) · [Review](#4-make-the-case-executable) ·
[Run](#5-check-your-current-code) · [Compare](#6-verify-a-beforeafter-fix) ·
[Troubleshoot](#troubleshooting)

## 1. Install the right binary

Download a binary from [Releases](https://github.com/keynertyc/kipuray/releases/latest).
Select the operating system **and** processor architecture:

| Your machine                               | Asset name ends with |
| ------------------------------------------ | -------------------- |
| Mac with Apple Silicon (M-series)          | `darwin_arm64`       |
| Mac with an Intel processor                | `darwin_amd64`       |
| Linux with an Intel/AMD x86-64 processor   | `linux_amd64`        |
| Linux with a 64-bit ARM processor          | `linux_arm64`        |
| Windows with an Intel/AMD x86-64 processor | `windows_amd64.exe`  |
| Windows with a 64-bit ARM processor        | `windows_arm64.exe`  |

On macOS or Linux, `uname -m` helps identify the architecture: `arm64`/`aarch64`
correspond to `arm64`; `x86_64` corresponds to `amd64`.

For example, after downloading v0.1.1 on an Apple Silicon Mac:

```sh
cd ~/Downloads
mv kipuray_v0.1.1_darwin_arm64 kipuray
chmod +x ./kipuray
./kipuray version
./kipuray doctor
```

Rename the downloaded file only if there is no existing `kipuray` file in that
directory. You can also keep the original filename and use it in the commands.
On Linux or an Intel Mac, substitute the matching downloaded filename.
On Windows, rename the file to `kipuray.exe` and use `.\kipuray.exe` in PowerShell
instead of `./kipuray`.

Chrome/Chromium must be installed for recording and execution. The downloaded
binary includes the workspace UI; you do not need Go, Node.js, or Docker to run it.
Your application still needs its own normal runtime and dependencies.

If you built Kipuray from source, use the executable at `bin/kipuray` instead.
Run the commands below from the directory containing the executable, use its
absolute path, or install it on your `PATH` and call `kipuray` directly. You do not
need to copy Kipuray into every application repository.

## 2. Start your application locally

In **Terminal A**, start your application and any test backend/database it needs.
Check that the page works at an HTTP or HTTPS loopback address, such as
`http://127.0.0.1:4173/checkout`.

**v0.1 compatibility:** choose a conventional DOM page using finite JSON Fetch/XHR
requests. Use a local build/preview, or a development mode with hot reload disabled.
Many development servers open a WebSocket for HMR; a recording containing that
traffic is currently marked non-executable. Kipuray's `dev` command does not turn
off your application's HMR or start its server for you.

For your first case, choose a public/test page that does not depend on an existing
login, cookies, or browser storage. Each execution uses a fresh browser profile;
v0.1 does not automatically restore authentication or record password-entry actions.

### Example: an existing Vite application

If your project already defines the usual `build` and `preview` scripts, run:

```sh
# Terminal A: inside your application repository
npm run build
npm run preview -- --host 127.0.0.1 --port 4173 --strictPort
```

This serves the built application locally rather than opening the HMR development
server. Rebuild after changing your source. Keep your test backend running and
ensure your preview's API URL/proxy configuration points to that backend during
capture. Other frameworks should use their equivalent local build/serve workflow.
See [Vite's local preview documentation](https://vite.dev/guide/static-deploy.html#testing-the-app-locally).

Kipuray's workspace uses port **4317** by default. The application being tested
uses its own port. The included demo additionally reserves **3000** and **3001**;
stop that demo with **Ctrl+C** before starting a conflicting application server.

## 3. Record the bug

In **Terminal B**, run the downloaded executable, replacing the example URL with
your application's actual starting page:

```sh
./kipuray dev http://127.0.0.1:4173/checkout --capture-json
```

`--capture-json` preselects JSON payload capture in the setup form. Only use it with
synthetic test data; review the data before exporting or sharing.

1. Kipuray opens its workspace and recording setup. Confirm the **Application URL**.
2. Enable **Include JSON API payloads** if your scenario needs API responses.
   Without them, you must fill in and review the missing fixtures before execution.
3. Click **Open recording window**. Interact with the dedicated Chrome window,
   rather than your normal browser tab.
4. Reproduce a short sequence ending at the bug. After typing, move focus to
   another field or click the next control so the input change is captured.
5. Return to the Kipuray workspace and click **Stop & review**. Leave Terminal B open.

The workspace normally opens at `http://127.0.0.1:4317`. If you use `--no-open`,
open the full **Workspace:** URL printed in Terminal B to preserve the target/setup
parameters. Recording starts when you click **Open recording window**.

## 4. Make the case executable

Open **Scenario & data**:

1. Give the case a recognizable title.
2. Under **Recorded actions**, replace each masked input with a synthetic test
   value. Review the selectors and action order.
3. Expand each entry in **Captured API data**. Check the request URL/body, HTTP
   status, and response JSON. Fill omitted values with suitable test data, then
   select **I reviewed this fixture and it contains only test data**.
4. Under **Expected result**, click **Add**. Describe the **correct** behavior,
   even if the broken version cannot show it yet.
5. Click **Save case**. Resolve any validation errors before running or exporting.

For a checkout bug, an expectation might be:

| Field         | Example                          |
| ------------- | -------------------------------- |
| Condition     | `Text includes`                  |
| CSS selector  | `[data-testid="coupon-message"]` |
| Expected text | `Coupon not valid`               |

Use a selector that belongs to **your** application. Stable IDs or test attributes
help keep the same scenario valid after a code change:

```html
<p data-testid="coupon-message" role="status">Coupon not valid</p>
```

You can also check that an element is visible or that the final URL path matches.
The **No uncaught errors** condition can supplement a functional expectation;
it is not sufficient on its own. A recording can be viewable while still lacking
the data needed to execute it.

## 5. Check your current code

1. Open **Verify fix**.
2. Uncheck **Compare the previous and corrected versions** for a single run.
3. Set **Target application** to your running local application's starting URL.
   Replace the demo's default port if it is not your app's port.
4. Leave **Scenario** set to **Original responses** and click **Run scenario**.

Kipuray opens your current application in headless Chrome, repeats the actions,
and supplies the saved API responses. Inspect **Execution history** for each check,
missing data, and browser evidence.

If the bug was captured correctly and your expectation describes its correction,
the broken version should fail that expectation. Change your code, rebuild/restart
the local app as needed, and run the **same case** again. A pass checks that scenario;
it does not establish a before/after comparison by itself.

Fixture-based execution checks frontend behavior against recorded API data. It
does not prove that a real backend update, database write, or external integration
works. Those need their own backend/integration tests.

## 6. Verify a before/after fix

Start the previous and corrected application versions at two different local
URLs, using separate working copies or your normal local preview workflow.
Both must already be running, without unsupported HMR/streaming traffic.

For example:

| Application                   | Starting URL                     |
| ----------------------------- | -------------------------------- |
| Previous version with the bug | `http://127.0.0.1:4173/checkout` |
| Corrected version             | `http://127.0.0.1:4174/checkout` |

In **Verify fix**, enable **Compare the previous and corrected versions**, replace
both default URLs, select the scenario, and click **Verify fix**. Use the same
functional expectation for both versions.

| Previous   | Corrected  | Comparison                                   |
| ---------- | ---------- | -------------------------------------------- |
| Failed     | Passed     | **Verified** for this scenario               |
| Passed     | Passed     | **Not reproduced** in the previous version   |
| Failed     | Failed     | **Still failing**                            |
| Passed     | Failed     | **Regression**                               |
| Incomplete | Any result | **Incomplete**; resolve the missing evidence |

Kipuray does not check out Git commits, start application servers, or prepare their
databases. Those remain part of your application's local workflow.

## 7. Explore a response variant

In **Scenario & data**, expand a captured response and click **Create a response
variant**. Give it a name, change its JSON, status, or delay, click **Add variant**,
and **Save case**. In **Verify fix**, select the variant in **Scenario** and run it.

For example, replace a successful response with HTTP `500` and
`{"message":"Temporarily unavailable"}` to check an error state.
The original response stays saved. Expectations currently belong to the whole
case; if a variant needs a different expected message, export/import a separate
copy first when you want to preserve both sets of expectations.

## 8. Keep the case for regression checks

Save your changes and click **Export .kpr**. The download uses the case ID as its
filename; the examples below assume you renamed it to `checkout.kpr`.

```sh
./kipuray open ./checkout.kpr
./kipuray run ./checkout.kpr --target http://127.0.0.1:4174/checkout
./kipuray compare ./checkout.kpr \
  --before http://127.0.0.1:4173/checkout \
  --after http://127.0.0.1:4174/checkout \
  --format junit --output result.xml
```

Stop the current workspace with **Ctrl+C** before using `open`, since it starts a
workspace on the same default port. The `run` and `compare` commands do not need
the workspace server; they read the file and execute the scenario directly.

Exit `0` means passed/verified, `1` means a failed check or a comparison that did
not verify the correction, and `2` means an incomplete or invalid execution.
Opening a case for visual playback can work without the original application;
running it still requires the target application and Chromium.

## Troubleshooting

| Symptom                                         | What to check                                                                                                                                                                                                                              |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Permission denied on macOS/Linux                | Run `chmod +x ./kipuray` on the downloaded executable.                                                                                                                                                                                     |
| Wrong architecture / bad CPU type               | Check the asset table above; Apple Silicon uses `darwin_arm64`, Intel Mac uses `darwin_amd64`.                                                                                                                                             |
| macOS blocks an unidentified developer          | Confirm you downloaded from this repository's release. Follow [Apple's instructions](https://support.apple.com/102445) for reviewing/allowing the executable in Privacy & Security.                                                        |
| Chrome cannot be found or started               | Install Chrome/Chromium. For a nonstandard location, pass `--chrome /absolute/path/to/browser` to `dev`, `demo`, `run`, or `compare`. For Linux sandbox errors, see [the hosting guide](hosting.md#chromium-on-linux-workstations-and-ci). |
| Port 4317 is in use                             | Stop the previous Kipuray terminal with Ctrl+C, or use `./kipuray dev http://127.0.0.1:4173 --listen 127.0.0.1:4318`. Open the workspace URL printed by that process.                                                                      |
| Port 3000/3001 is in use when starting the demo | Stop the conflicting app or the previous demo, or use your own app with `dev` on its existing port.                                                                                                                                        |
| Unsupported WebSocket or other browser API      | Use a local build/preview without HMR. WebSocket/SSE, workers, and service-worker-dependent scenarios are outside v0.1's supported execution scope.                                                                                        |
| The dedicated browser is logged out             | Fresh profiles do not inherit your usual cookies or storage. Start with an authentication-independent test page; password-entry actions are not recorded.                                                                                  |
| Run button is disabled                          | Add a functional expectation and review masked actions/fixtures. Interrupted or truncated recordings need a new complete capture.                                                                                                          |
| Missing fixture / unused API requests           | Check the starting route, synthetic values, query/request body, and action sequence against the capture. Use JSON Fetch/XHR fixtures; unexpected calls do not fall through to your backend.                                                |
| Element not found / ambiguous selector          | Check the selector against the current DOM. Prefer stable IDs/test attributes and preserve them across both versions.                                                                                                                      |
| Changes seem to have no effect                  | Rebuild/restart the target preview after editing source. Verify you are testing its actual URL and port, then save and rerun the case.                                                                                                     |
| Both versions pass                              | The original failure was not reproduced. Check that the previous version and expected result actually distinguish the bug.                                                                                                                 |

Keep the terminal serving your application and the terminal serving the workspace
open during interactive work. **Ctrl+C** in Kipuray's terminal stops its workspace
and active recording; it does not stop an independently started application server.

For a guided first check without your app, see the [bundled demo](../README.md#try-the-bundled-demo).
For what is captured or shared, see [data handling](data-handling.md).
