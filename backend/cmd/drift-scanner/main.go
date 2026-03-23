package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lume/backend/internal/adapters/repositories"
	"github.com/lume/backend/internal/core/services"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	ctx := context.Background()

	cfg, err := configFromEnv()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	log.Printf("Starting drift scan — workspace=%s layer=%s tf_workspace=%s ref=%s",
		cfg.workspaceID, cfg.layerID, cfg.tfWorkspace, cfg.gitRef)

	// ── Wire dependencies (no long-running server lifecycle needed for a job) ──
	cloner := repositories.NewGitCloner(cfg.gitToken)
	runner := repositories.NewTofuPlanRunner()

	repo, err := repositories.NewFirestoreWorkspaceRepository()
	if err != nil {
		log.Fatalf("failed to connect to Firestore: %v", err)
	}

	svc := services.NewDriftScannerService(cloner, runner, repo)

	result, err := svc.ScanLayer(ctx, services.ScanParams{
		WorkspaceID: cfg.workspaceID,
		LayerID:     cfg.layerID,
		TFWorkspace: cfg.tfWorkspace,
		RepoURL:     cfg.repoURL,
		GitRef:      cfg.gitRef,
		LayerPath:   cfg.layerPath,
	})
	if err != nil {
		log.Printf("ERROR: drift scan failed — %v", err)
		os.Exit(1)
	}

	log.Printf("Drift scan complete — status=%s add=%d change=%d destroy=%d",
		result.Status, result.AddCount, result.ChangeCount, result.DestroyCount)

	// Exit 1 if drifted so Cloud Run Job marks the execution as actionable.
	if result.Status == "drifted" {
		os.Exit(2) // non-zero but distinct from a hard error
	}
}

// config holds all runtime parameters, supplied entirely via environment variables.
// Secrets (GIT_TOKEN) are injected by Cloud Run from Secret Manager at execution time.
type config struct {
	workspaceID string // WORKSPACE_ID
	layerID     string // LAYER_ID
	tfWorkspace string // TF_WORKSPACE
	repoURL     string // GIT_REPO_URL  — e.g. https://github.com/org/repo.git
	gitRef      string // GIT_REF       — branch, tag, or commit SHA
	layerPath   string // GIT_LAYER_PATH — relative path in repo; defaults to "."
	gitToken    string // GIT_TOKEN      — PAT injected via Secret Manager
}

func configFromEnv() (*config, error) {
	var missing []string

	mustGet := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := &config{
		workspaceID: mustGet("WORKSPACE_ID"),
		layerID:     mustGet("LAYER_ID"),
		tfWorkspace: mustGet("TF_WORKSPACE"),
		repoURL:     mustGet("GIT_REPO_URL"),
		gitRef:      mustGet("GIT_REF"),
		gitToken:    mustGet("GIT_TOKEN"),
		layerPath:   os.Getenv("GIT_LAYER_PATH"), // optional
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	if cfg.layerPath == "" {
		cfg.layerPath = "."
	}

	return cfg, nil
}
