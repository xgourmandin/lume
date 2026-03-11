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
	Address   string `json:"address"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Instances []struct {
		Attributes struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Parent      string `json:"parent"`
			ProjectID   string `json:"project_id"`
			Number      string `json:"number"`
		} `json:"attributes"`
	} `json:"instances"`
}

func (p *TofuParser) Parse(ctx context.Context, stateData []byte) (*domain.Organization, error) {
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
				folders[inst.Attributes.ID] = &domain.Folder{
					ID:          inst.Attributes.ID,
					DisplayName: inst.Attributes.DisplayName,
					Parent:      inst.Attributes.Parent,
				}
			case "google_project":
				// If project_id is empty in attributes, might be using 'name' or another field
				pid := inst.Attributes.ProjectID
				if pid == "" {
					pid = inst.Attributes.ID
				}
				projects[inst.Attributes.ID] = &domain.Project{
					ID:          inst.Attributes.ID,
					ProjectID:   pid,
					DisplayName: inst.Attributes.DisplayName,
					Parent:      inst.Attributes.Parent,
				}
			}
		}
	}

	// Second pass: Reconstruct the hierarchy
	for _, folder := range folders {
		p.addToHierarchy(folder.Parent, folder, folders, orgs)
	}

	for _, project := range projects {
		p.addToHierarchy(project.Parent, project, folders, orgs)
	}

	// Find the root organization
	// Usually there's only one in a Landing Zone state
	if len(orgs) == 0 {
		return nil, fmt.Errorf("no organization found in state")
	}

	// Return the first org found (simplification)
	for _, org := range orgs {
		return org, nil
	}

	return nil, nil
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

	switch parentType {
	case "organizations":
		org, ok := orgs[parentID]
		if !ok {
			org = &domain.Organization{ID: parentID, DisplayName: "Organization " + parentID}
			orgs[parentID] = org
		}
		if f, ok := child.(*domain.Folder); ok {
			org.Folders = append(org.Folders, f)
		} else if pr, ok := child.(*domain.Project); ok {
			org.Projects = append(org.Projects, pr)
		}
	case "folders":
		folder, ok := folders[parentID]
		if ok {
			if f, ok := child.(*domain.Folder); ok {
				folder.Folders = append(folder.Folders, f)
			} else if pr, ok := child.(*domain.Project); ok {
				folder.Projects = append(folder.Projects, pr)
			}
		}
	}
}
