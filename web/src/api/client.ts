import type { HubSpotError } from './types';

const BASE_URL = '';  // empty = same origin, Vite proxy handles dev

export class ApiError extends Error {
  constructor(
    public status: number,
    public data: HubSpotError,
  ) {
    super(data.message);
    this.name = 'ApiError';
  }
}

export class NetworkError extends Error {
  constructor() {
    super('Network error — unable to reach the server');
    this.name = 'NetworkError';
  }
}

export async function apiFetch<T>(path: string, opts?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${BASE_URL}${path}`, {
      ...opts,
      headers: {
        'Content-Type': 'application/json',
        ...opts?.headers,
      },
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw err;
    }
    throw new NetworkError();
  }

  if (res.status === 204) return undefined as T;

  const data = await res.json();

  if (!res.ok) {
    throw new ApiError(res.status, data as HubSpotError);
  }

  return data as T;
}
