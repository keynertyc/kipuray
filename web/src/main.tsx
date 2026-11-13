import { CaptureDialog } from "./capture-dialog";
import React, {
  Suspense,
  lazy,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { createRoot } from "react-dom/client";
import {
  ArrowDownToLine,
  ArrowRight,
  ChevronRight,
  FilePlus2,
  FolderOpen,
  Link2,
  LoaderCircle,
  Radio,
  Square,
  Trash2,
  Unlink,
  Upload,
  X,
} from "lucide-react";
import { api, APIError, post } from "./api";
import type { BugCase, Summary } from "./types";
import { CaseWorkspace } from "./workspace";
import "./styles.css";

const Replay = lazy(() => import("./replay"));
function Mark() {
  return (
    <svg
      width="28"
      height="32"
      viewBox="0 0 28 32"
      fill="none"
      aria-hidden="true"
    >
      <path
        d="M7 2v28M21 2v28M7 9c0-7 14-7 14 0s-14 7-14 14 14 7 14 0"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
      />
    </svg>
  );
}
function App() {
  const sharedToken = location.pathname.startsWith("/shared/")
    ? location.pathname.split("/")[2]
    : "";
  const [workspaceDirty, setWorkspaceDirty] = useState(false);
  const [cases, setCases] = useState<Summary[]>([]),
    [current, setCurrent] = useState<BugCase | null>(null);
  const [local, setLocal] = useState(false),
    [recording, setRecording] = useState(false),
    [captureOpen, setCaptureOpen] = useState(false);
  const [target, setTarget] = useState(
    new URLSearchParams(location.search).get("target") ??
      "http://127.0.0.1:3000",
  );
  const [captureJSON, setCaptureJSON] = useState(
    new URLSearchParams(location.search).get("captureJSON") === "true",
  );
  const [busy, setBusy] = useState(""),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [login, setLogin] = useState(false),
    [token, setToken] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);
  const refresh = useCallback(
    async () => setCases(await api<Summary[]>("/cases")),
    [],
  );
  const openCase = useCallback(async (id: string) => {
    setError("");
    try {
      setCurrent(await api<BugCase>("/cases/" + id));
      history.replaceState(null, "", "/?case=" + id);
    } catch (e) {
      setError(message(e));
    }
  }, []);
  useEffect(() => {
    let active = true;
    async function load() {
      try {
        if (sharedToken) {
          const c = await api<BugCase>("/shared/" + sharedToken);
          if (active) setCurrent(c);
          return;
        }
        const status = await api<{ local: boolean; capture: string }>(
          "/status",
        );
        if (!active) return;
        setLocal(status.local);
        setRecording(status.capture !== "idle");
        await refresh();
        const params = new URLSearchParams(location.search);
        const id = params.get("case");
        if (id) await openCase(id);
        if (params.has("target")) setCaptureOpen(true);
      } catch (e) {
        if (e instanceof APIError && e.status === 401) setLogin(true);
        else setError(message(e));
      }
    }
    void load();
    return () => {
      active = false;
    };
  }, [sharedToken, refresh, openCase]);
  useEffect(() => {
    if (!notice) return;
    const timer = setTimeout(() => setNotice(""), 6000);
    return () => clearTimeout(timer);
  }, [notice]);
  async function task(label: string, fn: () => Promise<void>) {
    setBusy(label);
    setError("");
    try {
      await fn();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy("");
    }
  }
  async function save(c: BugCase) {
    const updated = await api<BugCase>("/cases/" + c.id, {
      method: "PUT",
      body: JSON.stringify(c),
    });
    setCurrent(updated);
    await refresh();
    return updated;
  }
  async function importFile(file?: File) {
    if (!file) return;
    await task("Importing case", async () => {
      const c = await api<BugCase>("/import", { method: "POST", body: file });
      setCurrent(c);
      history.replaceState(null, "", "/?case=" + c.id);
      await refresh();
      setNotice("Case imported. Review its data before running.");
    });
  }
  if (login)
    return (
      <main className="login">
        <div className="brand">
          <Mark />
          kipuray
        </div>
        <h1>Your debugging workspace.</h1>
        <p>Enter the owner token configured on this server.</p>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void task("Signing in", async () => {
              await post("/login", { token });
              setLogin(false);
              location.reload();
            });
          }}
        >
          <label>
            Workspace token
            <input
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              autoComplete="current-password"
              required
            />
          </label>
          <button className="primary" disabled={!!busy}>
            Sign in <ArrowRight size={16} />
          </button>
        </form>
        {error ? (
          <p role="alert" className="error">
            {error}
          </p>
        ) : null}
      </main>
    );
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <a className="brand" href="/">
          <Mark />
          <span>
            kipuray<small>LOCAL BUG REPLAY</small>
          </span>
        </a>
        <div className="workspace-label">
          <span className="connection-dot" />
          {sharedToken
            ? "Shared case"
            : local
              ? "Local workspace"
              : "Team workspace"}
        </div>
        {!sharedToken ? (
          <>
            <button
              className="primary new-record"
              onClick={() => setCaptureOpen(true)}
              disabled={!local || recording}
            >
              <Radio size={16} />
              New recording
            </button>
            <button
              className="quiet import-button"
              onClick={() => fileInput.current?.click()}
            >
              <Upload size={16} />
              Import a case
            </button>
            <input
              ref={fileInput}
              type="file"
              accept=".kpr"
              hidden
              onChange={(e) => {
                void importFile(e.target.files?.[0]);
                e.target.value = "";
              }}
            />
            <div className="section-label">
              YOUR CASES <span>{cases.length}</span>
            </div>
            <nav aria-label="Cases">
              {cases.map((c) => (
                <button
                  key={c.id}
                  className={
                    "case-link " + (current?.id === c.id ? "selected" : "")
                  }
                  onClick={() => void openCase(c.id)}
                >
                  <span className="case-node" />
                  <span>
                    <strong>{c.title}</strong>
                    <small>
                      {new Date(c.createdAt).toLocaleDateString(undefined, {
                        month: "short",
                        day: "numeric",
                      })}{" "}
                      · {c.actions} actions
                    </small>
                  </span>
                  <ChevronRight size={14} />
                </button>
              ))}
              {!cases.length ? (
                <p className="empty-sidebar">
                  Your first recording will appear here.
                </p>
              ) : null}
            </nav>
          </>
        ) : null}
        <div className="sidebar-footer">
          <span className="mini-knot" />
          Record. Reproduce. Resolve.
          <small>Private by default. Yours to share.</small>
          <a
            href="https://github.com/keynertyc/kipuray"
            target="_blank"
            rel="noreferrer"
          >
            Source & documentation <ArrowRight size={13} />
          </a>
        </div>
      </aside>
      <main className="main">
        <header className="topbar">
          <span>
            Workspace <ChevronRight size={13} />{" "}
            {current ? "Case details" : "Overview"}
          </span>
          <span className="local-tag">
            {sharedToken
              ? "Read-only link"
              : local
                ? "ON YOUR MACHINE"
                : "SELF-HOSTED"}
          </span>
        </header>
        {error ? (
          <div className="banner error" role="alert">
            {error}
            <button aria-label="Dismiss error" onClick={() => setError("")}>
              <X size={16} />
            </button>
          </div>
        ) : null}
        {notice ? (
          <div className="banner notice" role="status">
            {notice}
            <button aria-label="Dismiss notice" onClick={() => setNotice("")}>
              <X size={16} />
            </button>
          </div>
        ) : null}
        {recording ? (
          <div className="recording-bar" role="status">
            <span className="record-dot" />
            Recording in a dedicated browser. Reproduce the bug, then return
            here.
            <button
              disabled={!!busy}
              onClick={() =>
                void task("Saving recording", async () => {
                  const c = await post<BugCase>("/capture/stop");
                  setRecording(false);
                  setCurrent(c);
                  history.replaceState(null, "", "/?case=" + c.id);
                  await refresh();
                })
              }
            >
              <Square size={13} />
              Stop & review
            </button>
          </div>
        ) : null}
        {current ? (
          <>
            <div className="case-heading">
              <div>
                <h1>{current.title}</h1>
                <p className="origin">{current.origin}</p>
              </div>
              {!sharedToken ? (
                <div className="case-tools">
                  <a
                    className="button"
                    href={"/api/cases/" + current.id + "/download"}
                    aria-disabled={workspaceDirty}
                    onClick={(event) => {
                      if (workspaceDirty) {
                        event.preventDefault();
                        setNotice(
                          "Save your scenario changes before exporting.",
                        );
                      }
                    }}
                  >
                    <ArrowDownToLine size={15} />
                    Export .kpr
                  </a>
                  <button
                    onClick={() =>
                      void task("Creating share link", async () => {
                        if (workspaceDirty) {
                          setNotice(
                            "Save your scenario changes before sharing.",
                          );
                          return;
                        }
                        const result = await post<{ url: string }>(
                          "/cases/" + current.id + "/share",
                        );
                        const link = location.origin + result.url;
                        if (navigator.clipboard)
                          await navigator.clipboard
                            .writeText(link)
                            .catch(() => {});
                        setNotice("Read-only link (24 hours): " + link);
                      })
                    }
                  >
                    <Link2 size={15} />
                    Share
                  </button>
                  <button
                    aria-label="Revoke all sharing links"
                    title="Revoke sharing links"
                    onClick={() =>
                      void task("Revoking links", async () => {
                        await api("/cases/" + current.id + "/share", {
                          method: "DELETE",
                        });
                        setNotice("All links for this case have been revoked.");
                      })
                    }
                  >
                    <Unlink size={15} />
                  </button>
                  <button
                    aria-label="Delete case"
                    title="Delete case"
                    onClick={() => {
                      if (
                        confirm("Delete this case and its execution history?")
                      )
                        void task("Deleting case", async () => {
                          await api("/cases/" + current.id, {
                            method: "DELETE",
                          });
                          setCurrent(null);
                          history.replaceState(null, "", "/");
                          await refresh();
                        });
                    }}
                  >
                    <Trash2 size={15} />
                  </button>
                </div>
              ) : null}
            </div>
            <CaseWorkspace
              key={current.id}
              value={current}
              onSave={save}
              local={local}
              readOnly={!!sharedToken}
              sharedToken={sharedToken}
              onDirty={setWorkspaceDirty}
              onError={setError}
            />
          </>
        ) : (
          <div className="welcome">
            <h1>
              Give every bug
              <br />a way back.
            </h1>
            <p>
              Capture what happened. Recreate it against your local code.
              <br className="desktop-break" /> Check the fix with the same
              conditions.
            </p>
            <div className="welcome-actions">
              <button
                className="primary"
                disabled={!local || recording}
                onClick={() => setCaptureOpen(true)}
              >
                <Radio size={17} />
                Record a bug <ArrowRight size={16} />
              </button>
              <button onClick={() => fileInput.current?.click()}>
                <FolderOpen size={17} />
                Open a .kpr file
              </button>
            </div>
            <div className="workflow">
              <div>
                <span>01</span>
                <h2>Capture the context</h2>
                <p>
                  A short browser recording, actions, and the API responses that
                  matter.
                </p>
              </div>
              <div>
                <span>02</span>
                <h2>Make it repeatable</h2>
                <p>
                  Keep the original. Change a response. Explore the same
                  scenario.
                </p>
              </div>
              <div>
                <span>03</span>
                <h2>Verify the correction</h2>
                <p>
                  One expected result, checked against the broken and corrected
                  versions.
                </p>
              </div>
            </div>
            <div className="demo-hint">
              <FilePlus2 size={20} />
              <div>
                <strong>A real bug. A small first step.</strong>
                <p>
                  Run <code>kipuray demo</code> to try the included checkout.
                  Record “Apply coupon”, then compare ports 3000 and 3001.
                </p>
              </div>
            </div>
          </div>
        )}
        <footer className="main-footer">
          <span>Built for the part between a bug report and a fix.</span>
          <span>KIPURAY / OPEN SOURCE</span>
        </footer>
      </main>
      {captureOpen ? (
        <CaptureDialog
          target={target}
          setTarget={setTarget}
          captureJSON={captureJSON}
          setCaptureJSON={setCaptureJSON}
          busy={busy}
          local={local}
          error={error}
          onClose={() => setCaptureOpen(false)}
          onStart={() =>
            void task("Opening Chrome", async () => {
              await post("/capture/start", { url: target, captureJSON });
              setRecording(true);
              setCaptureOpen(false);
            })
          }
        />
      ) : null}
      {busy ? (
        <div className="busy" role="status">
          <LoaderCircle className="spin" size={16} />
          {busy}
        </div>
      ) : null}
    </div>
  );
}
function message(e: unknown) {
  return e instanceof Error ? e.message : String(e);
}
const isReplay = location.pathname.startsWith("/replay/");
createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    {isReplay ? (
      <Suspense
        fallback={<div className="replay-empty">Loading recording…</div>}
      >
        <Replay />
      </Suspense>
    ) : (
      <App />
    )}
  </React.StrictMode>,
);
