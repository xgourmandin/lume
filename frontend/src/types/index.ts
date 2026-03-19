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

/** Status of a workspace or layer as returned by the backend. */
export type SyncStatus = 'clean' | 'drifted' | 'error';

/** A named Terraform workspace within a layer (e.g. "default", "prod", "staging"). */
export interface TerraformWorkspace {
  id: string;        // terraform workspace name
  layer_id: string;
  status: SyncStatus;
  last_sync: string; // ISO-8601
}

/** A single Terraform state layer within a workspace. */
export interface Layer {
  id: string;
  name: string;
  last_sync: string; // ISO-8601 date string
  status: SyncStatus;
  /** Named Terraform workspaces within this layer. Defaults to [{id:"default"}] when absent. */
  workspaces?: TerraformWorkspace[];
}

/** Workspace metadata as returned by GET /api/v1/workspaces/:id */
export interface Workspace {
  id: string;
  last_sync: string; // ISO-8601 date string
  status: SyncStatus;
  layers: Layer[];
}

/** Payload for POST /api/v1/hierarchy/sync */
export interface SyncRequest {
  workspace_id: string;
  layer_id: string;
  /** Terraform workspace name (e.g. "default", "prod"). Defaults to "default" if omitted. */
  tf_workspace_id?: string;
  bucket: string;
  object: string;
}

/** Combined workspace detail: metadata + merged hierarchy */
export interface WorkspaceDetail {
  workspace: Workspace;
  hierarchy: Organization;
}
