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
const isLocalBackend = /^https?:\/\/(localhost|127\.0\.0\.1)/.test(BACKEND_URL);

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

/**
 * Returns the `Authorization` header needed to call the private backend, or an
 * empty object when running against a local backend.
 */
export async function backendAuthHeaders(): Promise<Record<string, string>> {
  if (isLocalBackend) {
    log('INFO', 'auth.skip', { reason: 'local-backend', backendUrl: BACKEND_URL });
    return {};
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

  const token = await res.text();
  log('INFO', 'auth.token_ok', { audience: BACKEND_URL, tokenLength: token.length });
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

  let res: Response;
  try {
    res = await fetch(url, {
      cache: 'no-store',
      ...init,
      headers: { ...(init.headers as Record<string, string>), ...authHeaders },
    });
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

  const data = await res.json();
  log(res.ok ? 'INFO' : 'ERROR', 'response.ok', { method, url, status: res.status, durationMs });
  return NextResponse.json(data, { status: res.status });
}
