import { record } from "rrweb";

// Runs only in a dedicated recording tab, injected before application scripts.
declare global {
  interface Window {
    __kipurayEmit: (payload: string) => void;
  }
}
if (window.top === window) {
  const send = (data: unknown) => {
    try {
      window.__kipurayEmit(
        JSON.stringify(data, (_key, value: unknown) => {
          if (typeof value !== "string" || !/^https?:\/\//i.test(value))
            return value;
          try {
            const url = new URL(value);
            for (const key of [...url.searchParams.keys()]) {
              if (
                /password|passwd|secret|token|authorization|cookie|api.?key|session.?id/i.test(
                  key,
                )
              )
                url.searchParams.set(key, "[REDACTED]");
            }
            url.username = "";
            url.password = "";
            return url.toString();
          } catch {
            return value;
          }
        }),
      );
    } catch {
      /* The recording may have stopped. */
    }
  };
  const selector = (el: Element): string => {
    for (const attr of ["data-testid", "data-test", "id"]) {
      const v = el.getAttribute(attr);
      if (v) {
        const q = `[${attr}=${JSON.stringify(v)}]`;
        if (document.querySelectorAll(q).length === 1) return q;
      }
    }
    const parts: string[] = [];
    let node: Element | null = el;
    while (node && node !== document.documentElement) {
      let part = node.tagName.toLowerCase();
      const parent: Element | null = node.parentElement;
      if (parent) {
        const siblings = Array.from(parent.children).filter(
          (x) => x.tagName === node!.tagName,
        );
        if (siblings.length > 1)
          part += `:nth-of-type(${siblings.indexOf(node) + 1})`;
      }
      parts.unshift(part);
      node = parent;
    }
    return parts.join(" > ");
  };
  const action = (
    kind: string,
    el: Element,
    value?: string,
    redacted = false,
  ) =>
    send({
      kind: "action",
      action: { kind, selector: selector(el), value, redacted, at: Date.now() },
    });
  document.addEventListener(
    "click",
    (e) => {
      const el =
        e.target instanceof Element
          ? e.target.closest('button,a,input,select,[role="button"]')
          : null;
      if (el) action("click", el);
    },
    true,
  );
  document.addEventListener(
    "change",
    (e) => {
      const el = e.target;
      if (
        el instanceof HTMLInputElement ||
        el instanceof HTMLTextAreaElement ||
        el instanceof HTMLSelectElement
      ) {
        if (
          el instanceof HTMLInputElement &&
          ["password", "hidden", "file", "checkbox", "radio"].includes(el.type)
        )
          return;
        const publicValue = el.hasAttribute("data-kipuray-public");
        action(
          el instanceof HTMLSelectElement ? "select" : "input",
          el,
          publicValue ? el.value : "",
          !publicValue,
        );
      }
    },
    true,
  );
  const start = () =>
    record({
      emit: (event) => send({ kind: "event", event }),
      maskAllInputs: true,
      blockSelector: "[data-private], [data-kipuray-block]",
      maskTextSelector: "[data-kipuray-mask]",
      inlineStylesheet: true,
      slimDOMOptions: { script: true, headFavicon: true },
      recordCanvas: false,
      collectFonts: false,
    });
  if (document.readyState === "loading")
    document.addEventListener("DOMContentLoaded", start, { once: true });
  else start();
}
