package repositories

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/lume/backend/internal/core/domain"
	"github.com/lume/backend/internal/core/ports"
)

// TofuPlanRunner executes tofu CLI commands to detect infrastructure drift.
// It expects the `tofu` binary to be available in PATH.
type TofuPlanRunner struct{}

// NewTofuPlanRunner returns a ports.PlanRunner backed by the system `tofu` binary.
func NewTofuPlanRunner() ports.PlanRunner {
	return &TofuPlanRunner{}
}

// RunPlan runs tofu init, selects (or creates) the given Terraform workspace,
// then executes `tofu plan -json -detailed-exitcode`. It always returns a
// non-nil *domain.DriftResult alongside any error.
func (r *TofuPlanRunner) RunPlan(ctx context.Context, workDir, tfWorkspace string) (*domain.DriftResult, error) {
	scannedAt := time.Now()

	// ── 1. tofu init ────────────────────────────────────────────────────────
	if err := r.streamCmd(ctx, workDir, "tofu", "init", "-input=false", "-no-color"); err != nil {
		return &domain.DriftResult{
			Status:       domain.DriftStatusError,
			ErrorMessage: fmt.Sprintf("tofu init failed: %s", err),
			ScannedAt:    scannedAt,
		}, fmt.Errorf("tofu init: %w", err)
	}

	// ── 2. tofu workspace select (fallback to new) ───────────────────────────
	if err := r.streamCmd(ctx, workDir, "tofu", "workspace", "select", tfWorkspace); err != nil {
		// Workspace may not exist yet — try to create it.
		if newErr := r.streamCmd(ctx, workDir, "tofu", "workspace", "new", tfWorkspace); newErr != nil {
			return &domain.DriftResult{
				Status:       domain.DriftStatusError,
				ErrorMessage: fmt.Sprintf("tofu workspace select/new %q failed: %s", tfWorkspace, newErr),
				ScannedAt:    scannedAt,
			}, fmt.Errorf("tofu workspace: %w", newErr)
		}
	}

	// ── 3. tofu plan ─────────────────────────────────────────────────────────
	return r.runPlan(ctx, workDir, scannedAt)
}

// streamCmd runs a non-plan tofu sub-command, streaming stdout/stderr directly
// to the process output so it appears in Cloud Run Job logs.
func (r *TofuPlanRunner) streamCmd(ctx context.Context, workDir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// planLine is a single JSON message in the `tofu plan -json` output stream.
type planLine struct {
	Type    string `json:"type"`
	Changes struct {
		Add    int `json:"add"`
		Change int `json:"change"`
		Remove int `json:"remove"`
	} `json:"changes"`
}

// runPlan executes `tofu plan -json -detailed-exitcode` and interprets the result.
//
// Exit code semantics:
//
//	0 → success, no changes (clean)
//	2 → success, changes present (drifted)
//	1 → error
func (r *TofuPlanRunner) runPlan(ctx context.Context, workDir string, scannedAt time.Time) (*domain.DriftResult, error) {
	cmd := exec.CommandContext(ctx,
		"tofu", "plan",
		"-json",
		"-detailed-exitcode",
		"-input=false",
		"-no-color",
	)
	cmd.Dir = workDir

	// Capture stdout for JSON parsing; tee stderr to the process output and a
	// buffer so we can include it in DriftResult.ErrorMessage if needed.
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	runErr := cmd.Run()

	exitCode := 0
	if runErr != nil {
		exitErr, ok := runErr.(*exec.ExitError)
		if !ok {
			// Binary not found or similar OS-level error.
			return &domain.DriftResult{
				Status:       domain.DriftStatusError,
				ErrorMessage: fmt.Sprintf("failed to execute tofu plan: %s", runErr),
				ScannedAt:    scannedAt,
			}, runErr
		}
		exitCode = exitErr.ExitCode()
	}

	result := &domain.DriftResult{ScannedAt: scannedAt}

	// Parse JSONL output to extract the change_summary message.
	scanner := bufio.NewScanner(&stdoutBuf)
	for scanner.Scan() {
		var line planLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type == "change_summary" {
			result.AddCount = line.Changes.Add
			result.ChangeCount = line.Changes.Change
			result.DestroyCount = line.Changes.Remove
		}
	}

	switch exitCode {
	case 0:
		result.Status = domain.DriftStatusClean
		return result, nil
	case 2:
		result.Status = domain.DriftStatusDrifted
		return result, nil
	default:
		// exit code 1 — tofu plan itself errored.
		result.Status = domain.DriftStatusError
		result.ErrorMessage = stderrBuf.String()
		return result, fmt.Errorf("tofu plan exited %d: %s", exitCode, stderrBuf.String())
	}
}
