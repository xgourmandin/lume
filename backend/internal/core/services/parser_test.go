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
	org, err := parser.Parse(context.Background(), stateData, "test-layer")

	assert.NoError(t, err)
	assert.NotNil(t, org)
	assert.Equal(t, "9876543210", org.ID)
	assert.Equal(t, "Organization 9876543210", org.DisplayName)

	// Check Engineering folder
	assert.Len(t, org.Folders, 1)
	engFolder := org.Folders[0]
	assert.Equal(t, "1234567890", engFolder.ID)
	assert.Equal(t, "Engineering", engFolder.DisplayName)
	assert.Equal(t, "test-layer", engFolder.LayerID)

	// Check Platforms folder
	assert.Len(t, engFolder.Folders, 1)
	platFolder := engFolder.Folders[0]
	assert.Equal(t, "2345678901", platFolder.ID)
	assert.Equal(t, "Platforms", platFolder.DisplayName)
	assert.Equal(t, "test-layer", platFolder.LayerID)

	// Check Lume Development project
	assert.Len(t, platFolder.Projects, 1)
	project := platFolder.Projects[0]
	assert.Equal(t, "lume-dev-project", project.ID)
	assert.Equal(t, "Lume Development", project.DisplayName)
	assert.Equal(t, "test-layer", project.LayerID)
}
