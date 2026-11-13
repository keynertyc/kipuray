export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message);
  }
}
export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch("/api" + path, {
    ...init,
    headers: {
      ...(init?.body instanceof Blob
        ? {}
        : { "Content-Type": "application/json" }),
      ...init?.headers,
    },
  });
  if (!response.ok) {
    const body = await response
      .json()
      .catch(() => ({ error: response.statusText }));
    throw new APIError(body.error, response.status);
  }
  return response.json() as Promise<T>;
}
export const post = <T>(path: string, value: unknown = {}) =>
  api<T>(path, { method: "POST", body: JSON.stringify(value) });
