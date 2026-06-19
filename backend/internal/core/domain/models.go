package domain

import (
	"sort"
	"strings"
	"time"
)

// Drift status constants shared across the domain.
const (
	DriftStatusClean   = "clean"
	DriftStatusDrifted = "drifted"
	DriftStatusError   = "error"
)

// DriftResult captures the outcome of a single `tofu plan` execution for one
// (workspaceID, layerID, tfWorkspaceID) tuple.
type DriftResult struct {
	Status       string    `json:"status"` // clean | drifted | error
	AddCount     int       `json:"add_count"`
	ChangeCount  int       `json:"change_count"`
	DestroyCount int       `json:"destroy_count"`
	ScannedAt    time.Time `json:"scanned_at"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// SyncAllResult captures the outcome of a bulk bucket synchronisation.
type SyncAllResult struct {
	Synced    int               `json:"synced"`
	Failed    int               `json:"failed"`
	Errors    []SyncObjectError `json:"errors,omitempty"`
	Hierarchy *Organization     `json:"hierarchy,omitempty"`
}

// SyncObjectError records which GCS object failed and why.
type SyncObjectError struct {
	Object string `json:"object"`
	Error  string `json:"error"`
}

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

// MergeFragments deep-merges a set of per-layer hierarchy fragments into a
// single, fully nested Organization.
//
// Each fragment is a flat (or partially nested) bag of folders and projects
// that may reference parents living in *other* layers. A single .tfstate rarely
// contains the whole chain from the organization down: the hierarchy layer owns
// the folders, while product/env layers own projects parented to those folders.
// MergeFragments therefore flattens every fragment, deduplicates nodes by ID,
// and re-derives the parent/child nesting from each node's Parent pointer across
// all layers at once. The result is deterministic regardless of fragment order.
func MergeFragments(frags []*Organization) *Organization {
	folderByID := make(map[string]*Folder)
	projectByKey := make(map[string]*Project)
	var folders []*Folder
	var projects []*Project

	for _, frag := range frags {
		if frag == nil {
			continue
		}
		ff, pp := frag.flatten()
		for _, f := range ff {
			if existing, ok := folderByID[f.ID]; ok {
				existing.mergeMeta(f)
				continue
			}
			folderByID[f.ID] = f
			folders = append(folders, f)
		}
		for _, p := range pp {
			key := projectKey(p)
			if existing, ok := projectByKey[key]; ok {
				existing.mergeMeta(p)
				continue
			}
			projectByKey[key] = p
			projects = append(projects, p)
		}
	}

	// Stable ordering so the rendered tree does not reshuffle between syncs.
	sort.SliceStable(folders, func(i, j int) bool { return folders[i].ID < folders[j].ID })
	sort.SliceStable(projects, func(i, j int) bool { return projectKey(projects[i]) < projectKey(projects[j]) })

	return AssembleHierarchy(folders, projects)
}

// AssembleHierarchy builds a nested Organization tree from flat folder and
// project lists, wiring children to parents via each node's Parent pointer.
// The organization ID is derived from any "organizations/<id>" parent reference;
// when none exists the root is a placeholder with an empty ID so that orphan
// folders and projects still surface in the tree rather than being dropped.
func AssembleHierarchy(folders []*Folder, projects []*Project) *Organization {
	folderByKey := make(map[string]*Folder, len(folders))
	for _, f := range folders {
		// Rebuild nesting from scratch; ignore any stale children carried over
		// from a previously stored (nested) fragment.
		f.Folders = nil
		f.Projects = nil
		folderByKey["folders/"+f.ID] = f
	}

	orgID := ""
	consider := func(parent string) {
		if rest, ok := strings.CutPrefix(parent, "organizations/"); ok && rest != "" {
			if orgID == "" || rest < orgID {
				orgID = rest
			}
		}
	}
	for _, f := range folders {
		consider(f.Parent)
	}
	for _, p := range projects {
		consider(p.Parent)
	}

	org := &Organization{ID: orgID}
	if orgID != "" {
		org.DisplayName = "Organization " + orgID
	}

	for _, f := range folders {
		if parent, ok := folderByKey[f.Parent]; ok && parent != f {
			parent.Folders = append(parent.Folders, f)
		} else {
			org.Folders = append(org.Folders, f)
		}
	}
	for _, p := range projects {
		if parent, ok := folderByKey[p.Parent]; ok {
			parent.Projects = append(parent.Projects, p)
		} else {
			org.Projects = append(org.Projects, p)
		}
	}

	return org
}

// flatten walks a (possibly nested) Organization fragment and returns every
// folder and project it contains as flat slices.
func (o *Organization) flatten() ([]*Folder, []*Project) {
	if o == nil {
		return nil, nil
	}
	var folders []*Folder
	projects := append([]*Project(nil), o.Projects...)
	for _, f := range o.Folders {
		collectFolder(f, &folders, &projects)
	}
	return folders, projects
}

func collectFolder(f *Folder, folders *[]*Folder, projects *[]*Project) {
	if f == nil {
		return
	}
	*folders = append(*folders, f)
	*projects = append(*projects, f.Projects...)
	for _, sub := range f.Folders {
		collectFolder(sub, folders, projects)
	}
}

func projectKey(p *Project) string {
	if p.ProjectID != "" {
		return p.ProjectID
	}
	return p.ID
}

// Folder represents a GCP Folder.
type Folder struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	Parent      string     `json:"parent"` // organizations/ID or folders/ID
	LayerID     string     `json:"layer_id,omitempty"`
	WorkspaceID string     `json:"workspace_id,omitempty"` // Terraform workspace name, e.g. "prod"
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

// mergeMeta fills in missing metadata on the receiver from a duplicate folder
// seen in another layer, without touching the children (rebuilt during assembly).
func (f *Folder) mergeMeta(other *Folder) {
	if f.DisplayName == "" {
		f.DisplayName = other.DisplayName
	}
	if f.Parent == "" {
		f.Parent = other.Parent
	}
	if f.LayerID == "" {
		f.LayerID = other.LayerID
	}
	if f.WorkspaceID == "" {
		f.WorkspaceID = other.WorkspaceID
	}
}

// Project represents a GCP Project.
type Project struct {
	ID          string      `json:"id"`
	ProjectID   string      `json:"project_id"`
	DisplayName string      `json:"display_name"`
	Parent      string      `json:"parent"` // organizations/ID or folders/ID
	LayerID     string      `json:"layer_id,omitempty"`
	WorkspaceID string      `json:"workspace_id,omitempty"` // Terraform workspace name, e.g. "prod"
	Resources   []*Resource `json:"resources,omitempty"`
}

// mergeMeta fills in missing metadata on the receiver from a duplicate project
// seen in another layer and unions any resources contributed by that layer.
func (p *Project) mergeMeta(other *Project) {
	if p.DisplayName == "" {
		p.DisplayName = other.DisplayName
	}
	if p.Parent == "" {
		p.Parent = other.Parent
	}
	if p.LayerID == "" {
		p.LayerID = other.LayerID
	}
	if p.WorkspaceID == "" {
		p.WorkspaceID = other.WorkspaceID
	}
	p.Resources = append(p.Resources, other.Resources...)
}

// Resource represents a leaf GCP resource managed by Terraform.
type Resource struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	ID          string `json:"id"` // Actual GCP Resource ID
	LayerID     string `json:"layer_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"` // Terraform workspace name, e.g. "prod"
}
