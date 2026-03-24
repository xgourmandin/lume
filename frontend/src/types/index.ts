export interface Resource {
  type: string;
  name: string;
  address: string;
  id: string;
  layer_id?: string;
  workspace_id?: string;
}

export interface Project {
  id: string;
  project_id: string;
  display_name: string;
  parent: string;
  layer_id?: string;
  workspace_id?: string;
  resources?: Resource[];
}

export interface Folder {
  id: string;
  display_name: string;
  parent: string;
  layer_id?: string;
  workspace_id?: string;
  folders?: Folder[];
  projects?: Project[];
}

export interface Organization {
  id: string;
  display_name: string;
  folders?: Folder[];
  projects?: Project[];
}

/** Status of a layer or tf-workspace as returned by the backend. */
export type SyncStatus = 'clean' | 'drifted' | 'error' | 'unknown';

/** A named Terraform workspace within a layer (e.g. "default", "prod", "staging"). */
export interface TerraformWorkspace {
  id: string;        // terraform workspace name
  layer_id: string;
  status: SyncStatus;
  last_sync?: string; // ISO-8601 — from the latest drift scan
}

/** A single Terraform state layer, built client-side from the hierarchy + drift results. */
export interface Layer {
  id: string;
  name: string;
  last_sync?: string; // most recent scanned_at across all tf-workspaces
  status: SyncStatus; // worst-case of all tf-workspace statuses
  workspaces?: TerraformWorkspace[];
}

/** Payload for POST /api/v1/hierarchy/sync */
export interface SyncRequest {
  layer_id: string;
  /** Terraform workspace name (e.g. "default", "prod"). Defaults to "default" if omitted. */
  tf_workspace_id?: string;
  bucket: string;
  object: string;
}

/** Result of a `tofu show -json` plan parsed by the backend drift scanner. */
export interface DriftResult {
  status: SyncStatus;
  add_count: number;
  change_count: number;
  destroy_count: number;
  scanned_at: string; // ISO-8601
  error_message?: string;
}
