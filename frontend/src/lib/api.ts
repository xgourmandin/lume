/**
 * Backend API client for Lume / GCP Terraform UI.
 *
 * All communication with the Go backend goes through this module so that
 * base-URL configuration, error handling and response typing live in one
 * place rather than being scattered across components.
 */

import type { Organization, Workspace, SyncRequest } from '@/types';

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const BASE_URL =
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, '') ?? 'http://localhost:8080';

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

