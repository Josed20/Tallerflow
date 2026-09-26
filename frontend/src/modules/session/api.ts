export class ApiError extends Error {
  constructor(public readonly status: number, message = 'No pudimos completar la solicitud.') {
    super(message)
  }
}

async function request<T>(path: string, init: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    credentials: 'include',
    headers: { Accept: 'application/json', ...(init.headers ?? {}) },
  })

  if (!response.ok) throw new ApiError(response.status)
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, body?: unknown, csrfToken?: string) =>
    request<T>(path, {
      method: 'POST',
      headers: {
        ...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
        ...(csrfToken === undefined ? {} : { 'X-CSRF-Token': csrfToken }),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
}
