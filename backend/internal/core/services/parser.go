package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/lume/backend/internal/core/domain"
	"github.com/lume/backend/internal/core/ports"
)

type TofuParser struct{}

func NewTofuParser() ports.StateParser {
	return &TofuParser{}
}

// tfState represents a simplified structure of the Tofu/Terraform JSON state.
type tfState struct {
	Resources []tfResource `json:"resources"`
}

type tfResource struct {
	Module    string `json:"module"`
	Address   string `json:"address"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Instances []struct {
		Attributes struct {
			ID          string `json:"id"`
			FolderID    string `json:"folder_id"`
			DisplayName string `json:"display_name"`
			Parent      string `json:"parent"`
			ProjectID   string `json:"project_id"`
			Number      string `json:"number"`
		} `json:"attributes"`
	} `json:"instances"`
}

// Parse converts raw Terraform state JSON into a GCP hierarchy fragment.
//
// A single .tfstate rarely contains the whole chain from the organization down:
// the hierarchy layer owns folders, while product/env layers own projects
// parented to folders that live in another layer. Parse therefore returns a
// *flat* fragment — every folder and project found in this state, each stamped
// with its layerID and tfWorkspaceID and carrying its Parent pointer. Nesting
// (and any organization root) is reconstructed across all layers later by
// domain.MergeFragments. A state with no organization-rooted node is a valid
// fragment, not an error.
func (p *TofuParser) Parse(_ context.Context, stateData []byte, layerID, tfWorkspaceID string) (*domain.Organization, error) {
	var state tfState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	folders := make(map[string]*domain.Folder)
	projects := make(map[string]*domain.Project)

	// First pass: collect all folders and projects
	for _, res := range state.Resources {
		for _, inst := range res.Instances {
			switch res.Type {
			case "google_folder":
				bareID := inst.Attributes.FolderID
				if bareID == "" {
					bareID = strings.TrimPrefix(inst.Attributes.ID, "folders/")
				}
				fullID := "folders/" + bareID
				folders[fullID] = &domain.Folder{
					ID:          bareID,
					DisplayName: inst.Attributes.DisplayName,
					Parent:      inst.Attributes.Parent,
					LayerID:     layerID,
					WorkspaceID: tfWorkspaceID,
				}
			case "google_project":
				pid := inst.Attributes.ProjectID
				if pid == "" {
					pid = strings.TrimPrefix(inst.Attributes.ID, "projects/")
				}
				projects[inst.Attributes.ID] = &domain.Project{
					ID:          strings.TrimPrefix(inst.Attributes.ID, "projects/"),
					ProjectID:   pid,
					DisplayName: inst.Attributes.DisplayName,
					Parent:      inst.Attributes.Parent,
					LayerID:     layerID,
					WorkspaceID: tfWorkspaceID,
				}
			}
		}
	}

	// Second pass: collect non-folder/non-project resources and attach them to projects.
	projectResources := make(map[string][]*domain.Resource) // keyed by project ID
	for _, res := range state.Resources {
		if res.Type == "google_folder" || res.Type == "google_project" {
			continue
		}
		for _, inst := range res.Instances {
			resource := &domain.Resource{
				Type:        res.Type,
				Name:        res.Name,
				Address:     res.Address,
				ID:          inst.Attributes.ID,
				LayerID:     layerID,
				WorkspaceID: tfWorkspaceID,
			}
			projKey := extractProjectKey(inst.Attributes.ID)
			projectResources[projKey] = append(projectResources[projKey], resource)
		}
	}

	// Attach resources to matching projects.
	for projPath, resources := range projectResources {
		if proj, ok := projects["projects/"+projPath]; ok {
			proj.Resources = append(proj.Resources, resources...)
		}
		// resources without a matching project are silently dropped.
	}

	// Emit a flat fragment in deterministic order. Cross-layer nesting and the
	// organization root are reconstructed by domain.MergeFragments at read time.
	fragment := &domain.Organization{}
	for _, folder := range folders {
		fragment.Folders = append(fragment.Folders, folder)
	}
	for _, project := range projects {
		fragment.Projects = append(fragment.Projects, project)
	}
	sort.Slice(fragment.Folders, func(i, j int) bool { return fragment.Folders[i].ID < fragment.Folders[j].ID })
	sort.Slice(fragment.Projects, func(i, j int) bool { return fragment.Projects[i].ProjectID < fragment.Projects[j].ProjectID })

	return fragment, nil
}

// extractProjectKey returns the project_id part from a resource ID like
// "projects/my-proj/..." returning "my-proj", or "" if not a project-scoped ID.
func extractProjectKey(resourceID string) string {
	if !strings.HasPrefix(resourceID, "projects/") {
		return ""
	}
	rest := strings.TrimPrefix(resourceID, "projects/")
	if idx := strings.Index(rest, "/"); idx != -1 {
		return rest[:idx]
	}
	return rest
}
