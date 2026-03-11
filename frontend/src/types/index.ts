export interface Resource {
  type: string;
  name: string;
  address: string;
  id: string;
}

export interface Project {
  id: string;
  project_id: string;
  display_name: string;
  parent: string;
  resources?: Resource[];
}

export interface Folder {
  id: string;
  display_name: string;
  parent: string;
  folders?: Folder[];
  projects?: Project[];
}

export interface Organization {
  id: string;
  display_name: string;
  folders?: Folder[];
  projects?: Project[];
}

export interface Workspace {
  id: string;
  last_sync: string;
  status: 'clean' | 'drifted' | 'error';
}
