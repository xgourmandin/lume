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

export const BACKEND_URL =
  process.env.API_URL?.replace(/\/$/, '') ?? 'http://localhost:3000';

const METADATA_IDENTITY_URL =
  'http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity';

// Local backends (dev) are unauthenticated and there is no metadata server.
const isLocalBackend = /^https?:\/\/(localhost|127\.0\.0\.1)/.test(BACKEND_URL);

/**
 * Returns the `Authorization` header needed to call the private backend, or an
 * empty object when running against a local backend.
 */
export async function backendAuthHeaders(): Promise<Record<string, string>> {
  if (isLocalBackend) return {};

  const res = await fetch(
    `${METADATA_IDENTITY_URL}?audience=${encodeURIComponent(BACKEND_URL)}`,
    { headers: { 'Metadata-Flavor': 'Google' }, cache: 'no-store' },
  );
  if (!res.ok) {
    throw new Error(
      `Failed to obtain ID token from metadata server: HTTP ${res.status}`,
    );
  }
  const token = await res.text();
  return { Authorization: `Bearer ${token}` };
}
