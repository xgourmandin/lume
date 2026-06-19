package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrganization_Merge_AddsNewFoldersAndProjects(t *testing.T) {
	base := &Organization{
		ID:          "org-1",
		DisplayName: "My Org",
		Folders: []*Folder{
			{ID: "f-1", DisplayName: "Engineering"},
		},
	}

	overlay := &Organization{
		ID: "org-1",
		Folders: []*Folder{
			{ID: "f-2", DisplayName: "Finance"},
		},
		Projects: []*Project{
			{ID: "p-1", ProjectID: "proj-root", DisplayName: "Root Project"},
		},
	}

	base.Merge(overlay)

	assert.Len(t, base.Folders, 2)
	assert.Len(t, base.Projects, 1)
	assert.Equal(t, "My Org", base.DisplayName, "existing display name must not be overwritten")
}

func TestOrganization_Merge_DeduplicatesProjects(t *testing.T) {
	base := &Organization{
		ID: "org-1",
		Projects: []*Project{
			{ProjectID: "proj-a", DisplayName: "Project A"},
		},
	}

	overlay := &Organization{
		ID: "org-1",
		Projects: []*Project{
			{ProjectID: "proj-a", DisplayName: "Project A (duplicate)"},
			{ProjectID: "proj-b", DisplayName: "Project B"},
		},
	}

	base.Merge(overlay)

	assert.Len(t, base.Projects, 2, "duplicate project should not be added")
	assert.Equal(t, "Project A", base.Projects[0].DisplayName, "original entry must be preserved")
}

func TestOrganization_Merge_RecursesMergeIntoExistingFolder(t *testing.T) {
	base := &Organization{
		ID: "org-1",
		Folders: []*Folder{
			{
				ID:          "f-1",
				DisplayName: "Engineering",
				Projects:    []*Project{{ProjectID: "proj-a"}},
			},
		},
	}

	overlay := &Organization{
		ID: "org-1",
		Folders: []*Folder{
			{
				ID: "f-1",
				Projects: []*Project{
					{ProjectID: "proj-b", DisplayName: "Project B"},
				},
			},
		},
	}

	base.Merge(overlay)

	assert.Len(t, base.Folders, 1, "no duplicate folder should be created")
	assert.Len(t, base.Folders[0].Projects, 2, "project from overlay must be added to existing folder")
}

func TestOrganization_Merge_NilOverlayIsNoop(t *testing.T) {
	base := &Organization{ID: "org-1", DisplayName: "My Org"}
	base.Merge(nil)
	assert.Equal(t, "My Org", base.DisplayName)
}

func TestMergeFragments_StitchesCrossLayerHierarchy(t *testing.T) {
	// Layer "hierarchy": folders only, rooted at the organization.
	hierarchy := &Organization{
		Folders: []*Folder{
			{ID: "1234567890", DisplayName: "Engineering", Parent: "organizations/9876543210", LayerID: "hierarchy"},
			{ID: "2345678901", DisplayName: "Platforms", Parent: "folders/1234567890", LayerID: "hierarchy"},
		},
	}
	// Layer "product-envs": a project parented to a folder owned by another layer.
	productEnvs := &Organization{
		Projects: []*Project{
			{ID: "lume-dev-project", ProjectID: "lume-dev-123", DisplayName: "Lume Development", Parent: "folders/2345678901", LayerID: "product-envs"},
		},
	}

	// Order must not matter: pass the project-only fragment first.
	merged := MergeFragments([]*Organization{productEnvs, hierarchy})

	// Organization root is derived from the org-rooted folder.
	assert.Equal(t, "9876543210", merged.ID)
	assert.Equal(t, "Organization 9876543210", merged.DisplayName)

	// Engineering nests under the org, Platforms under Engineering.
	assert.Len(t, merged.Folders, 1)
	eng := merged.Folders[0]
	assert.Equal(t, "Engineering", eng.DisplayName)
	assert.Len(t, eng.Folders, 1)
	plat := eng.Folders[0]
	assert.Equal(t, "Platforms", plat.DisplayName)

	// The cross-layer project lands under Platforms, not at the root.
	assert.Empty(t, merged.Projects)
	assert.Len(t, plat.Projects, 1)
	assert.Equal(t, "Lume Development", plat.Projects[0].DisplayName)
}

func TestMergeFragments_OrphansSurfaceUnderPlaceholderRoot(t *testing.T) {
	// No org-rooted node anywhere and the parent folder is unknown: the project
	// must still surface rather than being silently dropped.
	frag := &Organization{
		Projects: []*Project{
			{ID: "p", ProjectID: "orphan-123", Parent: "folders/does-not-exist"},
		},
	}

	merged := MergeFragments([]*Organization{frag})

	assert.Empty(t, merged.ID, "placeholder root has no organization id")
	assert.Len(t, merged.Projects, 1)
	assert.Equal(t, "orphan-123", merged.Projects[0].ProjectID)
}

func TestFolder_Merge_RecursiveSubFolders(t *testing.T) {
	base := &Folder{
		ID: "f-1",
		Folders: []*Folder{
			{ID: "f-1-1", DisplayName: "Sub A"},
		},
	}

	overlay := &Folder{
		ID: "f-1",
		Folders: []*Folder{
			{ID: "f-1-2", DisplayName: "Sub B"},
		},
	}

	base.Merge(overlay)

	assert.Len(t, base.Folders, 2)
}
