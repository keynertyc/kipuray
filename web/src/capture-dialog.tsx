import { useEffect, useRef } from "react";
import { LoaderCircle, Radio, X } from "lucide-react";

interface Props {
  target: string;
  setTarget: (value: string) => void;
  captureJSON: boolean;
  setCaptureJSON: (value: boolean) => void;
  busy: string;
  local: boolean;
  error: string;
  onClose: () => void;
  onStart: () => void;
}

export function CaptureDialog({
  target,
  setTarget,
  captureJSON,
  setCaptureJSON,
  busy,
  local,
  error,
  onClose,
  onStart,
}: Props) {
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const element = dialog.current;
    element?.showModal();
    return () => element?.close();
  }, []);
  return (
    <dialog
      className="modal"
      ref={dialog}
      aria-labelledby="capture-title"
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
    >
      <button
        className="close"
        aria-label="Close recording setup"
        disabled={!!busy}
        onClick={onClose}
      >
        <X size={19} />
      </button>
      <h2 id="capture-title">Start with your localhost.</h2>
      <p>
        A fresh Chrome window will open. Reproduce the bug, then stop the
        recording here.
      </p>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          onStart();
        }}
      >
        <label>
          Application URL
          <input
            type="url"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            required
            autoFocus
          />
        </label>
        <label className="checkbox">
          <input
            type="checkbox"
            checked={captureJSON}
            onChange={(e) => setCaptureJSON(e.target.checked)}
          />
          <span>
            Include JSON API payloads
            <strong>
              I’m using synthetic test data and will review it before sharing.
            </strong>
          </span>
        </label>
        <div className="helper">
          Inputs are masked by default. Replace masked values with test data in
          the scenario editor.
        </div>
        {error ? (
          <p className="error" role="alert">
            {error}
          </p>
        ) : null}
        <button className="primary wide" disabled={!!busy || !local}>
          {busy ? (
            <LoaderCircle className="spin" size={16} />
          ) : (
            <Radio size={16} />
          )}{" "}
          {busy || "Open recording window"}
        </button>
      </form>
    </dialog>
  );
}
