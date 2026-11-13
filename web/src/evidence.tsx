import { ChevronDown } from "lucide-react";
import { isComparison, type Evidence, type Run } from "./types";
export function EvidenceView({ result }: { result: Evidence }) {
  return (
    <details className={"evidence " + result.status} open>
      <summary>
        <span className={"result-status " + result.status}>
          {result.status.replaceAll("-", " ")}
        </span>
        <span>
          {isComparison(result) ? "Before / after comparison" : result.target}
        </span>
        <ChevronDown size={15} />
      </summary>
      {isComparison(result) ? (
        <div className="comparison-columns">
          <RunView result={result.before} label="Previous version" />
          <RunView result={result.after} label="Corrected version" />
        </div>
      ) : (
        <RunView result={result} label="Scenario execution" />
      )}
    </details>
  );
}
function RunView({ result: r, label }: { result: Run; label: string }) {
  return (
    <div className="run-view">
      <h4>
        {label}
        <span>{(r.durationMs / 1000).toFixed(1)} s</span>
      </h4>
      <small>{r.target}</small>
      {r.steps.map((s, i) => (
        <div className="result-step" key={i}>
          <span className={"status-dot " + s.status} />
          <span>
            {s.label}
            {s.detail ? <small>{s.detail}</small> : null}
          </span>
        </div>
      ))}
      {r.problems.map((p, i) => (
        <p className="problem" key={i}>
          {p}
        </p>
      ))}
      {r.screenshot ? (
        <details>
          <summary>Browser evidence</summary>
          <img
            className="evidence-image"
            alt={label + " browser screenshot"}
            src={"data:image/png;base64," + r.screenshot}
          />
        </details>
      ) : null}
    </div>
  );
}
