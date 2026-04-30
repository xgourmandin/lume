/**
 * Backend API client for Lume / GCP Terraform UI.
 *
 * All communication with the Go backend goes through the Next.js API route
 * handlers (/api/v1/...) which proxy requests to the backend using the
 * server-side `API_URL` environment variable.  No public/client-side env var
 * is required for routing — all fetch calls use relative URLs so they work
 * correctly in both SSR and client-side rendering contexts.
 */

import type { Organization, SyncRequest, SyncAllResult, DriftResult } from '@/types';

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// Always use the Next.js API routes (relative paths).  The server route
// handlers resolve the actual backend URL via the server-only `API_URL` env
// var, keeping it out of the client bundle entirely.
const BASE_URL = '/api/v1';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (body?.error) message = body.error;
    } catch {
      // ignore json parse error, use status text
    }
    throw new Error(message);
  }
  return res.json() as Promise<T>;
}

// ---------------------------------------------------------------------------
// Hierarchy endpoints
// ---------------------------------------------------------------------------

/**
 * Fetch the merged GCP hierarchy.
 * Proxied via Next.js → GET /api/v1/hierarchy → backend
 */
export async function fetchHierarchy(): Promise<Organization> {
  const res = await fetch(`${BASE_URL}/hierarchy`);
  return handleResponse<Organization>(res);
}

/**
 * Trigger a state sync for a specific layer.
 * Proxied via Next.js → POST /api/v1/hierarchy/sync → backend
 */
export async function syncWorkspace(payload: SyncRequest): Promise<Organization> {
  const res = await fetch(`${BASE_URL}/hierarchy/sync`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  return handleResponse<Organization>(res);
}

/**
 * Sync every .tfstate file found in the configured GCS bucket.
 * Proxied via Next.js → POST /api/v1/hierarchy/sync-all → backend
 */
export async function syncAllWorkspaces(): Promise<SyncAllResult> {
  const res = await fetch(`${BASE_URL}/hierarchy/sync-all`, {
    method: 'POST',
  });
  return handleResponse<SyncAllResult>(res);
}

// ---------------------------------------------------------------------------
// Drift result endpoints
// ---------------------------------------------------------------------------

// Mock drift results used as fallback when the backend is unreachable.
// Keys follow the same "{layerId}--{tfWorkspaceId}" convention used by Firestore.
const MOCK_DRIFT_RESULTS: Record<string, DriftResult> = {
  // ── org layer ─────────────────────────────────────────────────────────────
  'org--default': {
    status: 'clean',
    add_count: 0,
    change_count: 0,
    destroy_count: 0,
    scanned_at: new Date(Date.now() - 8 * 60 * 1000).toISOString(),
  },

  // ── network layer ─────────────────────────────────────────────────────────
  'network--default': {
    status: 'drifted',
    add_count: 1,
    change_count: 1,
    destroy_count: 0,
    scanned_at: new Date(Date.now() - 8 * 60 * 1000).toISOString(),
  },

  // ── security layer ────────────────────────────────────────────────────────
  'security--default': {
    status: 'clean',
    add_count: 0,
    change_count: 0,
    destroy_count: 0,
    scanned_at: new Date(Date.now() - 25 * 60 * 1000).toISOString(),
  },

  // ── projects layer ────────────────────────────────────────────────────────
  'projects--default': {
    status: 'error',
    add_count: 0,
    change_count: 0,
    destroy_count: 0,
    scanned_at: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString(),
    error_message:
      'Error: Failed to refresh state\n' +
      '  on main.tf line 23, in resource "google_project" "billing":\n' +
      '  Error: googleapi: Error 403: The caller does not have permission, forbidden\n\n' +
      'Make sure the scanner service account has roles/viewer at the org level.',
  },

  // ── apps layer ────────────────────────────────────────────────────────────
  'apps--default': {
    status: 'clean',
    add_count: 0,
    change_count: 0,
    destroy_count: 0,
    scanned_at: new Date(Date.now() - 12 * 60 * 60 * 1000).toISOString(),
  },
  'apps--prod': {
    status: 'drifted',
    add_count: 0,
    change_count: 2,
    destroy_count: 1,
    scanned_at: new Date(Date.now() - 13 * 60 * 60 * 1000).toISOString(),
  },
  'apps--staging': {
    status: 'clean',
    add_count: 0,
    change_count: 0,
    destroy_count: 0,
    scanned_at: new Date(Date.now() - 15 * 60 * 60 * 1000).toISOString(),
  },
  'apps--dev': {
    status: 'clean',
    add_count: 0,
    change_count: 0,
    destroy_count: 0,
    scanned_at: new Date(Date.now() - 20 * 60 * 60 * 1000).toISOString(),
  },
};

/**
 * Fetch the latest drift scan result for a (layerId, tfWorkspaceId) pair.
 * Proxied via Next.js → GET /api/v1/drift/{layerId}/{tfWorkspaceId} → backend
 *
 * Falls back to mock data when the backend is unreachable.
 */
export async function fetchDriftResult(
  layerId: string,
  tfWorkspaceId: string,
): Promise<DriftResult> {
  try {
    const res = await fetch(
      `${BASE_URL}/drift/${encodeURIComponent(layerId)}/${encodeURIComponent(tfWorkspaceId)}`,
    );
    return await handleResponse<DriftResult>(res);
  } catch {
    const key = `${layerId}--${tfWorkspaceId}`;
    const mock = MOCK_DRIFT_RESULTS[key];
    if (mock) return mock;
    throw new Error('Drift result not available');
  }
}
