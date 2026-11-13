export function path(raw: string) {
  try {
    const u = new URL(raw);
    return u.pathname + u.search;
  } catch {
    return raw;
  }
}
export function pretty(raw: string) {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}
