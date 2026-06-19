/**
 * Server-only helpers for proxying requests to the private backend Cloud Run
 * service.
 *
 * The backend rejects any request that does not carry a Google-signed ID token
 * (IAM `run.invoker` is restricted to the frontend service account).  Cloud Run
 * provides such a token from the instance metadata server, using the backend
 * service URL as the audience.  In local development the metadata server is not
 * reachable, so we simply skip the header.
 */

import { NextResponse } from 'next/server';

export const BACKEND_URL =
  process.env.API_URL?.replace(/\/$/, '') ?? 'http://localhost:3000';

const METADATA_IDENTITY_URL =
  'http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity';

// Local backends (dev) are unauthenticated and there is no metadata server.
// Parse the host rather than substring-matching the URL so that hostnames such
// as `localhost.attacker.example` are NOT treated as local (which would skip
// auth), and so IPv6 loopback / 0.0.0.0 dev backends ARE recognised.
const isLocalBackend = computeIsLocalBackend(BACKEND_URL);

function computeIsLocalBackend(urlStr: string): boolean {
  try {
    const host = new URL(urlStr).hostname.toLowerCase();
    return (
      host === 'localhost' ||
      host === '127.0.0.1' ||
      host === '0.0.0.0' ||
      host === '::1' ||
      host === '[::1]'
    );
  } catch {
    return false;
  }
}

// Structured log line — Cloud Run forwards stdout/stderr to Cloud Logging, and
// JSON payloads are parsed into the `jsonPayload` field so they are filterable.
function log(
  level: 'INFO' | 'ERROR',
  stage: string,
  fields: Record<string, unknown>,
) {
  const line = JSON.stringify({
    severity: level,
    component: 'backend-proxy',
    stage,
    ...fields,
  });
  if (level === 'ERROR') console.error(line);
  else console.log(line);
}

// Google ID tokens are valid for ~1h.  Cache the token (keyed by audience, which
// is fixed per process) so we don't hit the metadata server on every proxied
// request, refreshing a few minutes before expiry.
const TOKEN_REFRESH_MARGIN_MS = 5 * 60 * 1000;
const FALLBACK_TOKEN_TTL_MS = 55 * 60 * 1000;
let cachedToken: { value: string; expiresAt: number } | null = null;

// Returns the token's `exp` claim (ms epoch), or null if it can't be decoded.
function decodeJwtExpMs(token: string): number | null {
  const parts = token.split('.');
  if (parts.length < 2) return null;
  try {
    const payload = JSON.parse(
      Buffer.from(parts[1], 'base64url').toString('utf8'),
    );
    return typeof payload.exp === 'number' ? payload.exp * 1000 : null;
  } catch {
    return null;
  }
}

/**
 * Returns the `Authorization` header needed to call the private backend, or an
 * empty object when running against a local backend.  The token is cached in
 * memory until shortly before its expiry.
 */
export async function backendAuthHeaders(): Promise<Record<string, string>> {
  if (isLocalBackend) {
    log('INFO', 'auth.skip', { reason: 'local-backend', backendUrl: BACKEND_URL });
    return {};
  }

  const now = Date.now();
  if (cachedToken && now < cachedToken.expiresAt - TOKEN_REFRESH_MARGIN_MS) {
    return { Authorization: `Bearer ${cachedToken.value}` };
  }

  let res: Response;
  try {
    res = await fetch(
      `${METADATA_IDENTITY_URL}?audience=${encodeURIComponent(BACKEND_URL)}`,
      { headers: { 'Metadata-Flavor': 'Google' }, cache: 'no-store' },
    );
  } catch (err) {
    log('ERROR', 'auth.metadata_unreachable', {
      backendUrl: BACKEND_URL,
      error: err instanceof Error ? err.message : String(err),
    });
    throw err;
  }

  if (!res.ok) {
    const body = await res.text().catch(() => '<unreadable>');
    log('ERROR', 'auth.token_failed', { status: res.status, body: body.slice(0, 500) });
    throw new Error(`Failed to obtain ID token from metadata server: HTTP ${res.status}`);
  }

  const token = (await res.text()).trim();
  if (!token) {
    log('ERROR', 'auth.empty_token', { audience: BACKEND_URL });
    throw new Error('Metadata server returned an empty ID token');
  }

  const expiresAt = decodeJwtExpMs(token) ?? now + FALLBACK_TOKEN_TTL_MS;
  cachedToken = { value: token, expiresAt };
  log('INFO', 'auth.token_ok', {
    audience: BACKEND_URL,
    tokenLength: token.length,
    expiresAt: new Date(expiresAt).toISOString(),
  });
  return { Authorization: `Bearer ${token}` };
}

/**
 * Proxies a request to the backend, attaching the ID token and logging each
 * stage so failures are diagnosable from Cloud Logging.  Returns a NextResponse
 * mirroring the backend status; on transport/parse failure it returns 502 with
 * the underlying error message.
 */
export async function proxyBackend(
  path: string,
  init: RequestInit = {},
): Promise<NextResponse> {
  const url = `${BACKEND_URL}${path}`;
  const method = init.method ?? 'GET';
  const startedAt = Date.now();

  let authHeaders: Record<string, string>;
  try {
    authHeaders = await backendAuthHeaders();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    log('ERROR', 'request.auth_error', { method, url, message });
    return NextResponse.json({ error: message }, { status: 502 });
  }

  log('INFO', 'request.start', { method, url });

  // Build headers via the Headers constructor so any caller-supplied shape
  // (Record, Headers instance, or [key, value][] tuples) is preserved before
  // the auth header is layered on top.
  const headers = new Headers(init.headers);
  for (const [key, value] of Object.entries(authHeaders)) {
    headers.set(key, value);
  }

  let res: Response;
  try {
    res = await fetch(url, { cache: 'no-store', ...init, headers });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    log('ERROR', 'request.fetch_failed', {
      method,
      url,
      durationMs: Date.now() - startedAt,
      // err.cause often holds the real reason (ECONNREFUSED, ENOTFOUND, timeout)
      cause: err instanceof Error && err.cause ? String(err.cause) : undefined,
      message,
    });
    return NextResponse.json({ error: message }, { status: 502 });
  }

  const durationMs = Date.now() - startedAt;
  const contentType = res.headers.get('content-type') ?? '';

  // A non-JSON body almost always means we hit Cloud Run's auth/ingress front
  // (which replies with HTML), not the Go backend — surface it explicitly
  // instead of letting res.json() throw an opaque parse error.
  if (!contentType.includes('application/json')) {
    const body = await res.text().catch(() => '<unreadable>');
    log('ERROR', 'response.non_json', {
      method,
      url,
      status: res.status,
      durationMs,
      contentType,
      bodyPreview: body.slice(0, 500),
    });
    return NextResponse.json(
      { error: `Backend returned non-JSON response (HTTP ${res.status})` },
      { status: 502 },
    );
  }

  // Even with a JSON content-type the body can be empty or truncated (e.g. a
  // connection reset mid-stream), so guard the parse and mirror every other
  // failure path with a 502 rather than letting the rejection escape as a 500.
  let data: unknown;
  try {
    data = await res.json();
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    log('ERROR', 'response.parse_failed', {
      method,
      url,
      status: res.status,
      durationMs,
      contentType,
      message,
    });
    return NextResponse.json(
      { error: `Backend returned malformed JSON (HTTP ${res.status})` },
      { status: 502 },
    );
  }

  log(res.ok ? 'INFO' : 'ERROR', res.ok ? 'response.ok' : 'response.error', {
    method,
    url,
    status: res.status,
    durationMs,
  });
  return NextResponse.json(data, { status: res.status });
}
