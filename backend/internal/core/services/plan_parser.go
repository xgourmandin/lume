package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lume/backend/internal/core/domain"
	"github.com/lume/backend/internal/core/ports"
)

type TofuPlanParser struct{}

func NewTofuPlanParser() ports.PlanParser {
	return &TofuPlanParser{}
}

// tfPlan is a minimal representation of the JSON output produced by `tofu show -json <planfile>`.
// See: https://developer.hashicorp.com/terraform/internals/json-format#plan-representation
type tfPlan struct {
	ResourceChanges []tfPlanResourceChange `json:"resource_changes"`
}

type tfPlanResourceChange struct {
	Address string `json:"address"`
	Change  struct {
		// Actions is a sorted list of actions that will be taken on the resource.
		// Possible values: "no-op", "read", "create", "update", "delete".
		// Replace operations appear as ["delete","create"] or ["create","delete"].
		Actions []string `json:"actions"`
	} `json:"change"`
}

// ParseDrift parses a Terraform/OpenTofu JSON plan file and returns a DriftResult
// with status, add/change/destroy counts, and scanned_at all computed server-side.
//
// Status rules:
//   - "drifted" if any resource has a non-no-op change
//   - "clean"   if every resource is a no-op / read
//   - "error"   is never produced here; callers should wrap parse failures themselves
func (p *TofuPlanParser) ParseDrift(_ context.Context, planData []byte) (*domain.DriftResult, error) {
	var plan tfPlan
	if err := json.Unmarshal(planData, &plan); err != nil {
		return nil, fmt.Errorf("invalid plan JSON: %w", err)
	}

	var add, change, destroy int
	for _, rc := range plan.ResourceChanges {
		actions := rc.Change.Actions
		switch {
		case len(actions) == 0:
			// nothing
		case len(actions) == 1:
			switch actions[0] {
			case "create":
				add++
			case "update":
				change++
			case "delete":
				destroy++
				// "no-op" and "read" contribute nothing
			}
		default:
			// Replace operation: ["delete","create"] or ["create","delete"].
			// Counts as one addition and one destruction, matching the tofu CLI summary.
			add++
			destroy++
		}
	}

	status := domain.DriftStatusClean
	if add+change+destroy > 0 {
		status = domain.DriftStatusDrifted
	}

	return &domain.DriftResult{
		Status:       status,
		AddCount:     add,
		ChangeCount:  change,
		DestroyCount: destroy,
		ScannedAt:    time.Now().UTC(),
	}, nil
}
