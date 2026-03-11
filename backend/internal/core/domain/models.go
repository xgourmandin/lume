package domain

import "time"

// Organization represents the root of the GCP hierarchy.
type Organization struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	Folders     []*Folder  `json:"folders,omitempty"`
	Projects    []*Project `json:"projects,omitempty"`
}

// Merge deep-merges the overlay Organization into the receiver.
// Folders and Projects are matched by ID; existing entries are updated
// in-place (last-write-wins) and unknown entries are appended.
func (o *Organization) Merge(overlay *Organization) {
	if overlay == nil {
		return
	}
	if o.DisplayName == "" && overlay.DisplayName != "" {
		o.DisplayName = overlay.DisplayName
	}

	folderIdx := make(map[string]*Folder, len(o.Folders))
	for _, f := range o.Folders {
		folderIdx[f.ID] = f
	}
	for _, of := range overlay.Folders {
		if base, ok := folderIdx[of.ID]; ok {
			base.Merge(of)
		} else {
			o.Folders = append(o.Folders, of)
		}
	}

	projectIdx := make(map[string]*Project, len(o.Projects))
	for _, p := range o.Projects {
		projectIdx[p.ProjectID] = p
	}
	for _, op := range overlay.Projects {
		if _, ok := projectIdx[op.ProjectID]; !ok {
			o.Projects = append(o.Projects, op)
		}
	}
}

// Folder represents a GCP Folder.
type Folder struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	Parent      string     `json:"parent"` // organizations/ID or folders/ID
	LayerID     string     `json:"layer_id,omitempty"`
	Folders     []*Folder  `json:"folders,omitempty"`
	Projects    []*Project `json:"projects,omitempty"`
}

// Merge deep-merges the overlay Folder into the receiver, recursively.
func (f *Folder) Merge(overlay *Folder) {
	if overlay == nil {
		return
	}
	if f.DisplayName == "" && overlay.DisplayName != "" {
		f.DisplayName = overlay.DisplayName
	}

	folderIdx := make(map[string]*Folder, len(f.Folders))
	for _, sub := range f.Folders {
		folderIdx[sub.ID] = sub
	}
	for _, of := range overlay.Folders {
		if base, ok := folderIdx[of.ID]; ok {
			base.Merge(of)
		} else {
			f.Folders = append(f.Folders, of)
		}
	}

	projectIdx := make(map[string]*Project, len(f.Projects))
	for _, p := range f.Projects {
		projectIdx[p.ProjectID] = p
	}
	for _, op := range overlay.Projects {
		if _, ok := projectIdx[op.ProjectID]; !ok {
			f.Projects = append(f.Projects, op)
		}
	}
}

// Project represents a GCP Project.
type Project struct {
	ID          string      `json:"id"`
	ProjectID   string      `json:"project_id"`
	DisplayName string      `json:"display_name"`
	Parent      string      `json:"parent"` // organizations/ID or folders/ID
	LayerID     string      `json:"layer_id,omitempty"`
	Resources   []*Resource `json:"resources,omitempty"`
}

// Resource represents a leaf GCP resource managed by Terraform.
type Resource struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Address string `json:"address"`
	ID      string `json:"id"` // Actual GCP Resource ID
	LayerID string `json:"layer_id,omitempty"`
}

// Layer represents a single Terraform state layer within a workspace.
// A workspace is composed of one or more layers (e.g. "org", "network",
// "projects"), each backed by its own .tfstate file.
type Layer struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	LastSync time.Time `json:"last_sync"`
	Status   string    `json:"status"` // clean, drifted, error
}

// Workspace represents a Tofu/Terraform workspace metadata.
type Workspace struct {
	ID       string    `json:"id"`
	LastSync time.Time `json:"last_sync"`
	Status   string    `json:"status"` // clean, drifted, error
	Layers   []Layer   `json:"layers,omitempty"`
}
