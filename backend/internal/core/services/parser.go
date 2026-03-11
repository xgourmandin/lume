package services

import (
	"context"
	"encoding/json"
	"fmt"
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

func (p *TofuParser) Parse(ctx context.Context, stateData []byte, layerID string) (*domain.Organization, error) {
	var state tfState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	folders := make(map[string]*domain.Folder)
	projects := make(map[string]*domain.Project)
	orgs := make(map[string]*domain.Organization)

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
				}
			default:
				// Treat everything else as a generic resource attached to its project.
				if res.Type != "google_folder" && res.Type != "google_project" {
					// We'll handle generic resources in a second pass below.
				}
			}
		}
	}

	// Second pass: collect non-folder/non-project resources and attach them to projects.
	// We build a helper map from project resource address prefix to project.
	projectResources := make(map[string][]*domain.Resource) // keyed by project ID path
	for _, res := range state.Resources {
		if res.Type == "google_folder" || res.Type == "google_project" {
			continue
		}
		for _, inst := range res.Instances {
			resource := &domain.Resource{
				Type:    res.Type,
				Name:    res.Name,
				Address: res.Address,
				ID:      inst.Attributes.ID,
				LayerID: layerID,
			}
			// Try to determine the project from the resource ID (projects/<pid>/...)
			// or fall back to attaching to an "unscoped" bucket.
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

	// Third pass: Reconstruct the hierarchy
	for _, folder := range folders {
		p.addToHierarchy(folder.Parent, folder, folders, orgs)
	}

	for _, project := range projects {
		p.addToHierarchy(project.Parent, project, folders, orgs)
	}

	// Find the root organization
	if len(orgs) == 0 {
		return nil, fmt.Errorf("no organization found in state")
	}

	for _, org := range orgs {
		return org, nil
	}

	return nil, nil
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

func (p *TofuParser) addToHierarchy(parent string, child interface{}, folders map[string]*domain.Folder, orgs map[string]*domain.Organization) {
	if parent == "" {
		return
	}

	parts := strings.Split(parent, "/")
	if len(parts) < 2 {
		return
	}
	parentType := parts[0]
	parentID := parts[1]

	// Reconstruct the full parent path for map lookup (e.g. "folders/910383262608")
	fullParent := parentType + "/" + parentID

	switch parentType {
	case "organizations":
		org, ok := orgs[fullParent]
		if !ok {
			org = &domain.Organization{ID: parentID, DisplayName: "Organization " + parentID}
			orgs[fullParent] = org
		}
		if f, ok := child.(*domain.Folder); ok {
			org.Folders = append(org.Folders, f)
		} else if pr, ok := child.(*domain.Project); ok {
			org.Projects = append(org.Projects, pr)
		}
	case "folders":
		folder, ok := folders[fullParent]
		if ok {
			if f, ok := child.(*domain.Folder); ok {
				folder.Folders = append(folder.Folders, f)
			} else if pr, ok := child.(*domain.Project); ok {
				folder.Projects = append(folder.Projects, pr)
			}
		}
	}
}
