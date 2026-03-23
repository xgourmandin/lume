/**
 * Backend API client for Lume / GCP Terraform UI.
 *
 * All communication with the Go backend goes through this module so that
 * base-URL configuration, error handling and response typing live in one
 * place rather than being scattered across components.
 */

import type { Organization, Workspace, SyncRequest, DriftResult } from '@/types';

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
 * Fetch the merged GCP hierarchy for a workspace.
 * Corresponds to GET /api/v1/hierarchy/{workspaceId}
 *
 * Every node in the response carries a layer_id field. Use that to filter
 * the tree client-side without any extra round-trips.
 */
export async function fetchHierarchy(workspaceId: string): Promise<Organization> {
  const res = await fetch(`${BASE_URL}/api/v1/hierarchy/${encodeURIComponent(workspaceId)}`);
  return handleResponse<Organization>(res);
}

/**
 * Trigger a state sync for a specific workspace layer.
 * Corresponds to POST /api/v1/hierarchy/sync
 *
 * Returns the freshly merged Organization after the sync.
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
// Workspace endpoints
// ---------------------------------------------------------------------------

/**
 * Fetch workspace metadata (id, status, layers, last_sync).
 * Corresponds to GET /api/v1/workspaces/{workspaceId}
 */
export async function fetchWorkspace(workspaceId: string): Promise<Workspace> {
  const res = await fetch(`${BASE_URL}/api/v1/workspaces/${encodeURIComponent(workspaceId)}`);
  return handleResponse<Workspace>(res);
}

/**
 * Fetch all workspace summaries.
 * Corresponds to GET /api/v1/workspaces
 */
export async function fetchWorkspaces(): Promise<Workspace[]> {
  const res = await fetch(`${BASE_URL}/api/v1/workspaces`);
  return handleResponse<Workspace[]>(res);
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
 * Fetch the latest drift scan result for a (workspaceId, layerId, tfWorkspaceId) tuple.
 * Corresponds to GET /api/v1/workspaces/{workspaceId}/drift/{layerId}/{tfWorkspaceId}
 *
 * Falls back to mock data when the backend is unreachable.
 */
export async function fetchDriftResult(
  workspaceId: string,
  layerId: string,
  tfWorkspaceId: string,
): Promise<DriftResult> {
  try {
    const res = await fetch(
      `${BASE_URL}/api/v1/workspaces/${encodeURIComponent(workspaceId)}/drift/${encodeURIComponent(layerId)}/${encodeURIComponent(tfWorkspaceId)}`,
    );
    return await handleResponse<DriftResult>(res);
  } catch {
    const key = `${layerId}--${tfWorkspaceId}`;
    const mock = MOCK_DRIFT_RESULTS[key];
    if (mock) return mock;
    throw new Error('Drift result not available');
  }
}

