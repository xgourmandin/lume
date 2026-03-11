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
