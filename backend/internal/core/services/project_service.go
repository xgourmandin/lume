package services

import (
	"context"
	"fmt"

	"github.com/lume/backend/internal/core/ports"
)

type ProjectVendingService struct {
	git ports.GitProvider
}

func NewProjectVendingService(git ports.GitProvider) *ProjectVendingService {
	return &ProjectVendingService{git: git}
}

func (s *ProjectVendingService) CreateProjectRequest(ctx context.Context, projectName, folderID string) (string, error) {
	// 1. Generate .tf content from template
	tfContent := fmt.Sprintf(`
resource "google_project" "%s" {
  name       = "%s"
  project_id = "%s-%d"
  parent     = "folders/%s"
}
`, projectName, projectName, projectName, 12345, folderID)

	files := map[string]string{
		fmt.Sprintf("projects/%s.tf", projectName): tfContent,
	}

	// 2. Open PR
	prURL, err := s.git.CreatePullRequest(
		ctx,
		"lume-landing-zone",
		fmt.Sprintf("vending/%s", projectName),
		fmt.Sprintf("feat: vend new project %s", projectName),
		"Automated project request from Lume Console.",
		files,
	)

	return prURL, err
}
