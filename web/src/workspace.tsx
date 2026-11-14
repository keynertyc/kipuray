import { FixtureEditor } from "./fixture-editor";
import { EvidenceView } from "./evidence";
import { path } from "./format";
import { useEffect, useState } from "react";
import {
  Check,
  ChevronDown,
  Code2,
  FlaskConical,
  GitCompareArrows,
  LoaderCircle,
  Plus,
  Save,
  ShieldCheck,
  Trash2,
} from "lucide-react";
import { api, post } from "./api";
import { type BugCase, type Evidence, type Expectation } from "./types";

type Props = {
  value: BugCase;
  onSave: (c: BugCase) => Promise<BugCase>;
  local: boolean;
  readOnly: boolean;
  sharedToken: string;
  onError: (message: string) => void;
  onDirty: (dirty: boolean) => void;
};
export function CaseWorkspace({
  value,
  onSave,
  local,
  readOnly,
  sharedToken,
  onError,
  onDirty,
}: Props) {
  const [draft, setDraft] = useState(value),
    [tab, setTab] = useState<"recording" | "scenario" | "verify">("recording");
  const [saving, setSaving] = useState(false),
    [saved, setSaved] = useState(false),
    [running, setRunning] = useState(false),
    [results, setResults] = useState<Evidence[]>([]);
  const [target, setTarget] = useState("http://127.0.0.1:3001"),
    [before, setBefore] = useState("http://127.0.0.1:3000"),
    [compare, setCompare] = useState(true),
    [variantId, setVariantId] = useState("");
  useEffect(() => setDraft(value), [value]);
  useEffect(() => onDirty(draft !== value), [draft, value, onDirty]);
  useEffect(() => {
    if (readOnly) return;
    let active = true;
    api<Evidence[]>("/cases/" + value.id + "/runs")
      .then((r) => {
        if (active) setResults(r);
      })
      .catch((e) => onError(String(e)));
    return () => {
      active = false;
    };
  }, [value.id, readOnly, onError]);
  const missing =
    draft.actions.filter((a) => a.redacted).length +
    draft.fixtures.filter((f) => f.redacted).length;
  const ready =
    !draft.executionProblems?.length &&
    !draft.truncated &&
    draft.recordingState !== "recording" &&
    missing === 0 &&
    draft.expectations.some((e) => e.kind !== "no-errors");
  const change = (next: BugCase) => {
    setDraft(next);
    setSaved(false);
  };
  async function save() {
    setSaving(true);
    try {
      await onSave(draft);
      setSaved(true);
    } catch (e) {
      onError(String(e));
    } finally {
      setSaving(false);
    }
  }
  async function execute() {
    setRunning(true);
    try {
      await onSave(draft);
      const result = await post<Evidence>("/cases/" + draft.id + "/run", {
        target,
        before: compare ? before : "",
        variantId,
      });
      setResults((r) => [result, ...r]);
    } catch (e) {
      onError(String(e));
    } finally {
      setRunning(false);
    }
  }
  return (
    <>
      {draft.executionProblems?.length ? (
        <div className="banner error" role="alert">
          {draft.executionProblems.join(". ")}
        </div>
      ) : null}
      {draft.recordingState === "recording" || draft.truncated ? (
        <div className="banner error" role="alert">
          This recording was interrupted or exceeded its limit. You can inspect
          it, but need a complete new capture to execute.
        </div>
      ) : null}
      <div className="case-summary">
        <span>
          <strong>{draft.actions.length}</strong> actions
        </span>
        <span>
          <strong>{draft.fixtures.length}</strong> API responses
        </span>
        <span>
          <strong>{draft.diagnostics.length}</strong> diagnostics
        </span>
        <span className={"readiness " + (ready ? "ready" : "needs-review")}>
          <span className="status-dot" />
          {ready
            ? "Ready to run"
            : missing
              ? `${missing} values need review`
              : "Add an expected result"}
        </span>
      </div>
      <nav className="tabs" aria-label="Case views">
        {(["recording", "scenario", "verify"] as const).map((t) => (
          <button
            key={t}
            aria-current={tab === t ? "page" : undefined}
            onClick={() => setTab(t)}
          >
            {t === "recording"
              ? "Recording"
              : t === "scenario"
                ? "Scenario & data"
                : "Verify fix"}
            {t === "scenario" && missing ? <span>{missing}</span> : null}
          </button>
        ))}
      </nav>
      {tab === "recording" ? (
        <div className="recording-grid">
          <section className="playback-panel">
            <div className="panel-title">
              <h2>What happened</h2>
              <span>Visual playback</span>
            </div>
            <iframe
              className="replay-frame"
              title="Recorded browser session"
              src={
                sharedToken
                  ? "/replay/shared/" + sharedToken
                  : "/replay/" + draft.id
              }
              sandbox="allow-scripts allow-same-origin"
            />
            <div className="panel-footnote">
              Playback reconstructs the recorded page. Use Verify fix to execute
              your current code.
            </div>
          </section>
          <aside className="activity-panel">
            <h2>Case activity</h2>
            {draft.diagnostics.length ? (
              <div className="diagnostics">
                {draft.diagnostics.map((d, i) => (
                  <div className="diagnostic" key={i}>
                    <span className="error-node" />
                    <div>
                      <strong>
                        {d.kind === "error" ? "Browser error" : "Diagnostic"}
                      </strong>
                      <p>{d.message}</p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="muted">No browser errors captured.</p>
            )}
            <h3>Network responses</h3>
            {draft.fixtures.length ? (
              draft.fixtures.map((f, i) => (
                <details className="network-item" key={f.id}>
                  <summary>
                    <span
                      className={
                        "http-status " + (f.status >= 400 ? "bad" : "")
                      }
                    >
                      {f.status}
                    </span>
                    <span>
                      {f.method} {path(f.url)}
                    </span>
                    <ChevronDown size={13} />
                  </summary>
                  <pre>
                    {f.body || "Payload omitted. Review it in Scenario & data."}
                  </pre>
                  <small>
                    Occurrence {i + 1}
                    {f.redacted ? " · Needs review" : ""}
                  </small>
                </details>
              ))
            ) : (
              <p className="muted">No API responses captured.</p>
            )}
          </aside>
        </div>
      ) : null}
      {tab === "scenario" ? (
        <div className="scenario-layout">
          <section>
            <div className="section-heading">
              <div>
                <h2>A repeatable case.</h2>
                <p>Review the data, then describe what should happen.</p>
              </div>
              {!readOnly ? (
                <button
                  className="primary"
                  onClick={() => void save()}
                  disabled={saving}
                >
                  {saving ? (
                    <LoaderCircle className="spin" size={15} />
                  ) : saved ? (
                    <Check size={15} />
                  ) : (
                    <Save size={15} />
                  )}{" "}
                  {saved ? "Saved" : "Save case"}
                </button>
              ) : null}
            </div>
            <label>
              Case title
              <input
                value={draft.title}
                maxLength={200}
                disabled={readOnly}
                onChange={(e) => change({ ...draft, title: e.target.value })}
              />
            </label>
            <h3>
              Recorded actions{" "}
              <span className="count">{draft.actions.length}</span>
            </h3>
            <div className="action-list">
              {draft.actions.map((a, i) => (
                <div className="action-row" key={i}>
                  <span className="step-number">{i + 1}</span>
                  <div>
                    <strong>{a.kind}</strong>
                    <code>{a.selector || a.value}</code>
                    {a.kind === "input" || a.kind === "select" ? (
                      <label className="action-value">
                        Synthetic test value
                        <input
                          value={a.value ?? ""}
                          placeholder={
                            a.redacted ? "Replace the masked value" : ""
                          }
                          disabled={readOnly}
                          onChange={(e) =>
                            change({
                              ...draft,
                              actions: draft.actions.map((v, n) =>
                                n === i
                                  ? {
                                      ...v,
                                      value: e.target.value,
                                      redacted: false,
                                    }
                                  : v,
                              ),
                            })
                          }
                        />
                        {a.redacted ? (
                          <span className="review-label">
                            Required before execution
                          </span>
                        ) : null}
                      </label>
                    ) : null}
                  </div>
                  {!readOnly ? (
                    <button
                      aria-label={"Remove action " + (i + 1)}
                      onClick={() =>
                        change({
                          ...draft,
                          actions: draft.actions.filter((_, n) => i !== n),
                        })
                      }
                    >
                      <Trash2 size={14} />
                    </button>
                  ) : null}
                </div>
              ))}
              {!draft.actions.length ? (
                <p className="muted">
                  No actions. A page-load scenario can still have expectations.
                </p>
              ) : null}
            </div>
            <div className="section-heading">
              <div>
                <h3>Expected result</h3>
                <p>
                  Describe the correct behavior, even if the broken version
                  cannot show it.
                </p>
              </div>
              {!readOnly ? (
                <button
                  onClick={() =>
                    change({
                      ...draft,
                      expectations: [
                        ...draft.expectations,
                        { kind: "text", selector: "", value: "" },
                      ],
                    })
                  }
                >
                  <Plus size={15} />
                  Add
                </button>
              ) : null}
            </div>
            {draft.expectations.map((e, i) => (
              <div className="expectation-row" key={i}>
                <label>
                  Condition
                  <select
                    value={e.kind}
                    disabled={readOnly}
                    onChange={(ev) =>
                      updateExpectation(i, { ...e, kind: ev.target.value })
                    }
                  >
                    <option value="text">Text includes</option>
                    <option value="visible">Element visible</option>
                    <option value="url">URL path equals</option>
                    <option value="no-errors">No uncaught errors</option>
                  </select>
                </label>
                {e.kind === "text" || e.kind === "visible" ? (
                  <label>
                    CSS selector
                    <input
                      value={e.selector ?? ""}
                      disabled={readOnly}
                      placeholder={'[data-testid="message"]'}
                      onChange={(ev) =>
                        updateExpectation(i, {
                          ...e,
                          selector: ev.target.value,
                        })
                      }
                    />
                  </label>
                ) : null}
                {e.kind === "text" || e.kind === "url" ? (
                  <label>
                    {e.kind === "url" ? "Path and query" : "Expected text"}
                    <input
                      value={e.value ?? ""}
                      disabled={readOnly}
                      onChange={(ev) =>
                        updateExpectation(i, { ...e, value: ev.target.value })
                      }
                    />
                  </label>
                ) : null}
                {!readOnly ? (
                  <button
                    aria-label="Remove expectation"
                    onClick={() =>
                      change({
                        ...draft,
                        expectations: draft.expectations.filter(
                          (_, n) => i !== n,
                        ),
                      })
                    }
                  >
                    <Trash2 size={14} />
                  </button>
                ) : null}
              </div>
            ))}
            <h3>Captured API data</h3>
            <p className="muted">
              Only include synthetic data. Mark a response reviewed after
              replacing omitted or sensitive values.
            </p>
            {draft.fixtures.map((f) => (
              <FixtureEditor
                key={f.id}
                fixture={f}
                readOnly={readOnly}
                onChange={(next) =>
                  change({
                    ...draft,
                    fixtures: draft.fixtures.map((v) =>
                      v.id === next.id ? next : v,
                    ),
                  })
                }
                onVariant={(v) => {
                  change({ ...draft, variants: [...draft.variants, v] });
                  setVariantId(v.id);
                }}
              />
            ))}
            {draft.variants.length ? (
              <>
                <h3>Response variants</h3>
                {draft.variants.map((v) => (
                  <div className="variant-row" key={v.id}>
                    <FlaskConical size={16} />
                    <span>
                      {v.name}
                      <small>
                        HTTP {v.status} · {v.delayMs} ms
                      </small>
                    </span>
                    {!readOnly ? (
                      <button
                        aria-label={"Delete variant " + v.name}
                        onClick={() => {
                          change({
                            ...draft,
                            variants: draft.variants.filter(
                              (x) => x.id !== v.id,
                            ),
                          });
                          if (variantId === v.id) setVariantId("");
                        }}
                      >
                        <Trash2 size={14} />
                      </button>
                    ) : null}
                  </div>
                ))}
              </>
            ) : null}
          </section>
          <aside className="review-note">
            <ShieldCheck size={25} />
            <h3>
              Useful evidence.
              <br />
              Deliberate sharing.
            </h3>
            <p>
              Review page text, payloads, and diagnostics before exporting.
              Masking cannot identify every kind of sensitive data.
            </p>
            {draft.warnings.map((w, i) => (
              <p key={i}>{w}</p>
            ))}
          </aside>
        </div>
      ) : null}
      {tab === "verify" ? (
        <div className="verify-layout">
          <section>
            <div className="section-heading">
              <div>
                <h2>Put the fix to the test.</h2>
                <p>Run the same conditions against your local application.</p>
              </div>
              <GitCompareArrows size={28} />
            </div>
            <div className="verify-explanation">
              Checks frontend behavior with recorded API responses. Your target
              applications must already be running.
            </div>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                void execute();
              }}
            >
              <label className="checkbox">
                <input
                  type="checkbox"
                  checked={compare}
                  onChange={(e) => setCompare(e.target.checked)}
                />
                <span>Compare the previous and corrected versions</span>
              </label>
              <div className="target-inputs">
                {compare ? (
                  <label>
                    Previous version
                    <input
                      type="url"
                      value={before}
                      required
                      onChange={(e) => setBefore(e.target.value)}
                    />
                  </label>
                ) : null}
                <label>
                  {compare ? "Corrected version" : "Target application"}
                  <input
                    type="url"
                    value={target}
                    required
                    onChange={(e) => setTarget(e.target.value)}
                  />
                </label>
              </div>
              <label>
                Scenario
                <select
                  value={variantId}
                  onChange={(e) => setVariantId(e.target.value)}
                >
                  <option value="">Original responses</option>
                  {draft.variants.map((v) => (
                    <option value={v.id} key={v.id}>
                      {v.name}
                    </option>
                  ))}
                </select>
              </label>
              {!ready ? (
                <p className="review-label">
                  {missing
                    ? "Review masked inputs and omitted responses in Scenario & data."
                    : "Add at least one functional expected result in Scenario & data."}
                </p>
              ) : null}
              <button
                className="primary"
                disabled={running || !local || readOnly || !ready}
              >
                {running ? (
                  <LoaderCircle size={17} className="spin" />
                ) : (
                  <GitCompareArrows size={17} />
                )}{" "}
                {running
                  ? "Checking the scenario…"
                  : compare
                    ? "Verify fix"
                    : "Run scenario"}
              </button>
              {!local ? (
                <p className="muted">
                  Download the case and run it on your workstation with the
                  Kipuray CLI.
                </p>
              ) : null}
            </form>
            <h3>Execution history</h3>
            {results.length ? (
              results.map((result, i) => (
                <EvidenceView key={i} result={result} />
              ))
            ) : (
              <div className="empty-results">
                <Code2 size={24} />
                <p>
                  Your results will appear here, with evidence for every check.
                </p>
              </div>
            )}
          </section>
          <aside className="review-note">
            <h3>What “verified” means</h3>
            <p>
              The previous version fails your expectation. The corrected version
              passes it under the same recorded conditions.
            </p>
            <p>
              If the previous version also passes, the original failure was not
              reproduced. Missing fixtures or broken selectors produce an
              incomplete result.
            </p>
          </aside>
        </div>
      ) : null}
    </>
  );
  function updateExpectation(index: number, e: Expectation) {
    change({
      ...draft,
      expectations: draft.expectations.map((v, i) => (i === index ? e : v)),
    });
  }
}
