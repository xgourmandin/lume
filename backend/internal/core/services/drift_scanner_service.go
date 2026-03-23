package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lume/backend/internal/core/domain"
	"github.com/lume/backend/internal/core/ports"
)

// ScanParams holds all configuration needed for a single drift scan execution.
// Each Cloud Run Job invocation is parameterised by exactly one set of these values,
// covering one (layerID, TFWorkspace) pair.
type ScanParams struct {
	WorkspaceID string // Lume workspace identifier
	LayerID     string // Terraform layer name, e.g. "org", "network", "projects"
	TFWorkspace string // Terraform workspace name, e.g. "prod", "staging"
	RepoURL     string // HTTPS URL of the git repository
	GitRef      string // branch, tag, or commit SHA to check out
	LayerPath   string // relative path within the repo to the layer directory; "." for root
}

// DriftScannerService orchestrates a single drift scan:
//  1. Clone the git repository into a temporary directory.
//  2. Navigate to the layer sub-directory.
//  3. Run tofu init + workspace select + plan.
//  4. Persist the DriftResult to the repository.
type DriftScannerService struct {
	cloner ports.CodeCloner
	runner ports.PlanRunner
	repo   ports.WorkspaceRepository
}

// NewDriftScannerService constructs a DriftScannerService with the supplied adapters.
func NewDriftScannerService(cloner ports.CodeCloner, runner ports.PlanRunner, repo ports.WorkspaceRepository) *DriftScannerService {
	return &DriftScannerService{
		cloner: cloner,
		runner: runner,
		repo:   repo,
	}
}

// ScanLayer executes a full drift scan for the (workspaceID, layerID, TFWorkspace)
// triple described by params. It always attempts to persist the result to Firestore,
// even when the plan itself errors out.
func (s *DriftScannerService) ScanLayer(ctx context.Context, params ScanParams) (*domain.DriftResult, error) {
	// Create a parent temp directory; git will create the repo sub-directory inside it.
	tmpParent, err := os.MkdirTemp("", "drift-scanner-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpParent) }()

	repoDir := filepath.Join(tmpParent, "repo")

	// ── 1. Clone ──────────────────────────────────────────────────────────────
	if err := s.cloner.CloneLayer(ctx, params.RepoURL, params.GitRef, repoDir); err != nil {
		result := &domain.DriftResult{
			Status:       domain.DriftStatusError,
			ErrorMessage: fmt.Sprintf("git clone failed: %s", err),
			ScannedAt:    time.Now(),
		}
		// Best-effort save so Firestore shows "error" rather than stale data.
		_ = s.repo.SaveDriftResult(ctx, params.WorkspaceID, params.LayerID, params.TFWorkspace, result)
		return result, err
	}

	// ── 2. Resolve the layer working directory ────────────────────────────────
	workDir := repoDir
	if params.LayerPath != "" && params.LayerPath != "." {
		workDir = filepath.Join(repoDir, params.LayerPath)
	}

	// ── 3. Run tofu plan ──────────────────────────────────────────────────────
	result, planErr := s.runner.RunPlan(ctx, workDir, params.TFWorkspace)
	if result == nil {
		// Defensive: RunPlan contract guarantees non-nil, but guard anyway.
		result = &domain.DriftResult{
			Status:    domain.DriftStatusError,
			ScannedAt: time.Now(),
		}
		if planErr != nil {
			result.ErrorMessage = planErr.Error()
		}
	}

	// ── 4. Persist ────────────────────────────────────────────────────────────
	if saveErr := s.repo.SaveDriftResult(ctx, params.WorkspaceID, params.LayerID, params.TFWorkspace, result); saveErr != nil {
		// Wrap both errors so the caller has the full picture.
		if planErr != nil {
			return result, fmt.Errorf("plan error: %w; additionally failed to persist result: %s", planErr, saveErr)
		}
		return result, fmt.Errorf("failed to persist drift result: %w", saveErr)
	}

	return result, planErr
}
