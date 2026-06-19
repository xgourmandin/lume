package services

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTofuParser_Parse(t *testing.T) {
	stateData, err := os.ReadFile("test_state.json")
	if err != nil {
		t.Fatalf("failed to read test state file: %v", err)
	}

	parser := NewTofuParser()
	frag, err := parser.Parse(context.Background(), stateData, "test-layer", "default")

	assert.NoError(t, err)
	assert.NotNil(t, frag)

	// Parse returns a FLAT fragment: every folder/project carries its Parent
	// pointer; nesting is reconstructed later by domain.MergeFragments. The
	// fragment is not org-rooted, so there is no synthesized organization here.
	assert.Empty(t, frag.ID)

	// Folders are emitted flat, sorted by ID.
	assert.Len(t, frag.Folders, 2)

	engFolder := frag.Folders[0]
	assert.Equal(t, "1234567890", engFolder.ID)
	assert.Equal(t, "Engineering", engFolder.DisplayName)
	assert.Equal(t, "organizations/9876543210", engFolder.Parent)
	assert.Equal(t, "test-layer", engFolder.LayerID)
	assert.Equal(t, "default", engFolder.WorkspaceID)
	assert.Empty(t, engFolder.Folders, "fragment folders must be flat, not nested")

	platFolder := frag.Folders[1]
	assert.Equal(t, "2345678901", platFolder.ID)
	assert.Equal(t, "Platforms", platFolder.DisplayName)
	assert.Equal(t, "folders/1234567890", platFolder.Parent)

	// Project is emitted flat with its folder parent pointer intact.
	assert.Len(t, frag.Projects, 1)
	project := frag.Projects[0]
	assert.Equal(t, "lume-dev-project", project.ID)
	assert.Equal(t, "Lume Development", project.DisplayName)
	assert.Equal(t, "folders/2345678901", project.Parent)
	assert.Equal(t, "test-layer", project.LayerID)
	assert.Equal(t, "default", project.WorkspaceID)
}

// TestTofuParser_NoOrganizationIsNotAnError guards the bug where a layer that
// contains no organization-rooted node (e.g. a product/env layer whose projects
// are parented to folders defined in another layer) failed to sync entirely.
func TestTofuParser_NoOrganizationIsNotAnError(t *testing.T) {
	state := `{"resources":[{"type":"google_project","name":"app","instances":[{"attributes":{"id":"app-prj","project_id":"app-prj-123","display_name":"App","parent":"folders/555"}}]}]}`

	parser := NewTofuParser()
	frag, err := parser.Parse(context.Background(), []byte(state), "product-envs", "lume")

	assert.NoError(t, err)
	assert.NotNil(t, frag)
	assert.Len(t, frag.Projects, 1)
	assert.Equal(t, "folders/555", frag.Projects[0].Parent)
}
