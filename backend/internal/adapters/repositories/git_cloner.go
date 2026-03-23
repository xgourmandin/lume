package repositories

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"github.com/lume/backend/internal/core/ports"
)

// GitCloner clones a private git repository using a personal access token
// injected into the HTTPS URL (oauth2:<token>@host). The token is sourced
// from the GIT_TOKEN environment variable, delivered via Secret Manager at
// Cloud Run Job execution time.
type GitCloner struct {
	token string
}

// NewGitCloner returns a ports.CodeCloner backed by the system `git` binary.
// token is the personal / service-account access token for the git host.
func NewGitCloner(token string) ports.CodeCloner {
	return &GitCloner{token: token}
}

// CloneLayer performs a shallow single-branch clone of repoURL at ref into destDir.
// destDir must not exist; it will be created by git.
func (g *GitCloner) CloneLayer(ctx context.Context, repoURL, ref, destDir string) error {
	authedURL, err := g.injectToken(repoURL)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx,
		"git", "clone",
		"--depth", "1",
		"--branch", ref,
		"--single-branch",
		authedURL,
		destDir,
	)

	// Stream git output to the process logs; capture stderr for error reporting.
	var stderrBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w — %s", err, stderrBuf.String())
	}
	return nil
}

// injectToken builds an authenticated HTTPS URL by embedding the token as the
// password with the "oauth2" username, which is accepted by GitHub, GitLab, and
// Bitbucket.
//
//	https://github.com/org/repo.git
//	→ https://oauth2:<token>@github.com/org/repo.git
func (g *GitCloner) injectToken(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid repo URL %q: %w", rawURL, err)
	}
	u.User = url.UserPassword("oauth2", g.token)
	return u.String(), nil
}
