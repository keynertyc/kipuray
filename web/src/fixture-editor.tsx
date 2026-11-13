import { useState } from "react";
import { ChevronDown, FlaskConical, Plus } from "lucide-react";
import type { Fixture, Variant } from "./types";
import { path } from "./format";
export function FixtureEditor({
  fixture: f,
  readOnly,
  onChange,
  onVariant,
}: {
  fixture: Fixture;
  readOnly: boolean;
  onChange: (f: Fixture) => void;
  onVariant: (v: Variant) => void;
}) {
  const [variant, setVariant] = useState(false),
    [body, setBody] = useState(f.body),
    [status, setStatus] = useState(f.status),
    [delay, setDelay] = useState(0),
    [name, setName] = useState(""),
    [error, setError] = useState("");
  return (
    <details className="fixture-editor">
      <summary>
        <span className={"http-status " + (f.status >= 400 ? "bad" : "")}>
          {f.status}
        </span>
        <code>
          {f.method} {path(f.url)}
        </code>
        {f.redacted ? (
          <span className="review-label">Review required</span>
        ) : (
          <span className="muted">Reviewed</span>
        )}
        <ChevronDown size={14} />
      </summary>
      <div className="fixture-content">
        <label>
          Request URL
          <input
            disabled={readOnly}
            value={f.url}
            onChange={(e) => onChange({ ...f, url: e.target.value })}
          />
        </label>
        <label>
          Response body
          <textarea
            disabled={readOnly}
            spellCheck={false}
            rows={7}
            value={f.body}
            onChange={(e) => onChange({ ...f, body: e.target.value })}
          />
        </label>
        <div className="field-pair">
          <label>
            HTTP status
            <input
              type="number"
              min="100"
              max="599"
              value={f.status}
              disabled={readOnly}
              onChange={(e) =>
                onChange({ ...f, status: Number(e.target.value) })
              }
            />
          </label>
          <label>
            Request body (if any)
            <input
              value={f.requestBody ?? ""}
              disabled={readOnly}
              onChange={(e) => onChange({ ...f, requestBody: e.target.value })}
            />
          </label>
        </div>
        {!readOnly ? (
          <>
            <label className="checkbox">
              <input
                type="checkbox"
                checked={!f.redacted}
                onChange={(e) =>
                  onChange({ ...f, redacted: !e.target.checked })
                }
              />
              <span>
                I reviewed this fixture and it contains only test data.
              </span>
            </label>
            <button
              onClick={() => {
                setBody(f.body);
                setStatus(f.status);
                setVariant(!variant);
              }}
            >
              <FlaskConical size={15} />
              Create a response variant
            </button>
            {variant ? (
              <div className="variant-editor">
                <h4>One change. Same scenario.</h4>
                <label>
                  Variant name
                  <input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="API unavailable"
                  />
                </label>
                <div className="field-pair">
                  <label>
                    Status
                    <input
                      type="number"
                      min="100"
                      max="599"
                      value={status}
                      onChange={(e) => setStatus(Number(e.target.value))}
                    />
                  </label>
                  <label>
                    Delay (ms)
                    <input
                      type="number"
                      min="0"
                      max="30000"
                      value={delay}
                      onChange={(e) => setDelay(Number(e.target.value))}
                    />
                  </label>
                </div>
                <label>
                  Replacement JSON
                  <textarea
                    rows={5}
                    spellCheck={false}
                    value={body}
                    onChange={(e) => setBody(e.target.value)}
                  />
                </label>
                {error ? (
                  <p className="error" role="alert">
                    {error}
                  </p>
                ) : null}
                <button
                  onClick={() => {
                    try {
                      JSON.parse(body);
                      if (
                        !name.trim() ||
                        status < 100 ||
                        status > 599 ||
                        delay < 0 ||
                        delay > 30000
                      )
                        throw new Error(
                          "Enter a name, valid HTTP status, and delay between 0 and 30000 ms.",
                        );
                      onVariant({
                        id: crypto.randomUUID().replaceAll("-", ""),
                        name: name.trim(),
                        fixtureId: f.id,
                        body,
                        status,
                        delayMs: delay,
                      });
                      setVariant(false);
                      setError("");
                    } catch (e) {
                      setError(String(e));
                    }
                  }}
                >
                  <Plus size={14} />
                  Add variant
                </button>
              </div>
            ) : null}
          </>
        ) : null}
      </div>
    </details>
  );
}
