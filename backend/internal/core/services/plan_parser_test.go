package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/lume/backend/internal/core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makePlan(actions ...[]string) []byte {
	type change struct {
		Actions []string `json:"actions"`
	}
	type rc struct {
		Address string `json:"address"`
		Change  change `json:"change"`
	}
	type plan struct {
		ResourceChanges []rc `json:"resource_changes"`
	}

	p := plan{}
	for i, a := range actions {
		p.ResourceChanges = append(p.ResourceChanges, rc{
			Address: "resource." + string(rune('a'+i)),
			Change:  change{Actions: a},
		})
	}
	b, _ := json.Marshal(p)
	return b
}

func TestTofuPlanParser_Clean(t *testing.T) {
	parser := NewTofuPlanParser()
	result, err := parser.ParseDrift(context.Background(), makePlan(
		[]string{"no-op"},
		[]string{"read"},
	))
	require.NoError(t, err)
	assert.Equal(t, domain.DriftStatusClean, result.Status)
	assert.Equal(t, 0, result.AddCount)
	assert.Equal(t, 0, result.ChangeCount)
	assert.Equal(t, 0, result.DestroyCount)
	assert.False(t, result.ScannedAt.IsZero())
}

func TestTofuPlanParser_Drifted(t *testing.T) {
	parser := NewTofuPlanParser()
	result, err := parser.ParseDrift(context.Background(), makePlan(
		[]string{"create"},
		[]string{"update"},
		[]string{"delete"},
	))
	require.NoError(t, err)
	assert.Equal(t, domain.DriftStatusDrifted, result.Status)
	assert.Equal(t, 1, result.AddCount)
	assert.Equal(t, 1, result.ChangeCount)
	assert.Equal(t, 1, result.DestroyCount)
}

func TestTofuPlanParser_Replace(t *testing.T) {
	// replace counts as +1 add and +1 destroy
	parser := NewTofuPlanParser()
	result, err := parser.ParseDrift(context.Background(), makePlan(
		[]string{"delete", "create"},
	))
	require.NoError(t, err)
	assert.Equal(t, domain.DriftStatusDrifted, result.Status)
	assert.Equal(t, 1, result.AddCount)
	assert.Equal(t, 0, result.ChangeCount)
	assert.Equal(t, 1, result.DestroyCount)
}

func TestTofuPlanParser_InvalidJSON(t *testing.T) {
	parser := NewTofuPlanParser()
	result, err := parser.ParseDrift(context.Background(), []byte("not-json"))
	assert.Nil(t, result)
	assert.Error(t, err)
}
