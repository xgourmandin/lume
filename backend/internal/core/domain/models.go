package domain

import "time"

// Organization represents the root of the GCP hierarchy.
type Organization struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	Folders     []*Folder  `json:"folders,omitempty"`
	Projects    []*Project `json:"projects,omitempty"`
}

// Folder represents a GCP Folder.
type Folder struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	Parent      string     `json:"parent"` // organizations/ID or folders/ID
	Folders     []*Folder  `json:"folders,omitempty"`
	Projects    []*Project `json:"projects,omitempty"`
}

// Project represents a GCP Project.
type Project struct {
	ID          string      `json:"id"`
	ProjectID   string      `json:"project_id"`
	DisplayName string      `json:"display_name"`
	Parent      string      `json:"parent"` // organizations/ID or folders/ID
	Resources   []*Resource `json:"resources,omitempty"`
}

// Resource represents a leaf GCP resource managed by Terraform.
type Resource struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Address string `json:"address"`
	ID      string `json:"id"` // Actual GCP Resource ID
}

// Workspace represents a Tofu/Terraform workspace metadata.
type Workspace struct {
	ID       string    `json:"id"`
	LastSync time.Time `json:"last_sync"`
	Status   string    `json:"status"` // clean, drifted, error
}
