/**
 * Backend API client for Lume / GCP Terraform UI.
 *
 * All communication with the Go backend goes through this module so that
 * base-URL configuration, error handling and response typing live in one
 * place rather than being scattered across components.
 */

import type { Organization, SyncRequest, DriftResult } from '@/types';

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const BASE_URL =
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, '') ?? 'http://localhost:3000';

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
 * Corresponds to GET /api/v1/hierarchy
 */
export async function fetchHierarchy(): Promise<Organization> {
  const res = await fetch(`${BASE_URL}/api/v1/hierarchy`);
  return handleResponse<Organization>(res);
}

/**
 * Trigger a state sync for a specific layer.
 * Corresponds to POST /api/v1/hierarchy/sync
 */
export async function syncWorkspace(payload: SyncRequest): Promise<Organization> {
  const res = await fetch(`${BASE_URL}/api/v1/hierarchy/sync`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  return handleResponse<Organization>(res);
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
 * Corresponds to GET /api/v1/drift/{layerId}/{tfWorkspaceId}
 *
 * Falls back to mock data when the backend is unreachable.
 */
export async function fetchDriftResult(
  layerId: string,
  tfWorkspaceId: string,
): Promise<DriftResult> {
  try {
    const res = await fetch(
      `${BASE_URL}/api/v1/drift/${encodeURIComponent(layerId)}/${encodeURIComponent(tfWorkspaceId)}`,
    );
    return await handleResponse<DriftResult>(res);
  } catch {
    const key = `${layerId}--${tfWorkspaceId}`;
    const mock = MOCK_DRIFT_RESULTS[key];
    if (mock) return mock;
    throw new Error('Drift result not available');
  }
}
