# Hosting the sharing workspace

The sharing workspace needs one process and a writable data directory. It stores,
imports, displays, exports, and shares cases. Recording and execution remain local
capabilities and are disabled for non-loopback listeners.

```sh
export KIPURAY_TOKEN="$(openssl rand -hex 32)"
kipuray serve --listen 0.0.0.0:4317 --data-dir /var/lib/kipuray --no-open
```

The owner token must contain at least 32 characters. Keep it in your service
manager's secret/environment configuration. Sign in through the web interface.
Restarting the process invalidates login cookies; changing the token and restarting
rotates owner access. Read-only sharing links remain until revoked or expired.

## Docker

```sh
export KIPURAY_TOKEN="$(openssl rand -hex 32)"
docker compose up --build -d
```

The included image runs as a non-root user and contains no automated browser.
The named volume stores `kipuray.db` and its SQLite journal files. The Compose
configuration binds the host port to `127.0.0.1`.

## HTTPS

Terminate HTTPS at a reverse proxy. For example, with Caddy on the same host:

```caddyfile
bugs.example.com {
    reverse_proxy 127.0.0.1:4317
}
```

Forward the original Host header and `X-Forwarded-Proto: https`. Do not cache case
responses or authentication pages. The application sets same-site, HttpOnly login
cookies and rejects cross-origin mutations. External access should go through the
HTTPS endpoint; the workspace itself serves HTTP behind the proxy.

## Backup and restore

For a straightforward consistent backup, stop the process and copy the entire
data directory. With Docker, stop the service and back up the named volume. Restore
the directory/volume with the same ownership before restarting. Do not copy only
the live database while leaving its WAL behind.

This version is intended for a developer or small trusted team using a single
workspace owner. It does not provide organization roles, SSO, distributed storage,
or remote execution of contributed code. Remove unneeded cases periodically to
bound disk usage; deleting a case also removes its run history and sharing links.

## Chromium on Linux workstations and CI

Some Ubuntu releases restrict unprivileged user namespaces through AppArmor.
If Chrome reports “No usable sandbox”, configure the distribution's browser
AppArmor profile or user-namespace support with your administrator. Kipuray does
not automatically disable Chrome's sandbox. The ephemeral GitHub Actions runner
explicitly enables user namespaces before its browser integration tests. The
sharing-only Docker server does not launch a browser and is unaffected.

See [Chromium's AppArmor guidance](https://chromium.googlesource.com/chromium/src/+/main/docs/security/apparmor-userns-restrictions.md).
