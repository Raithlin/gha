package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CurrentBranch returns the checked-out branch without changing repository
// state. A detached HEAD is rejected because it is not a safe PR head default.
func CurrentBranch(ctx context.Context, workdir string) (string, error) {
	command := exec.CommandContext(ctx, "git", "branch", "--show-current")
	command.Dir = workdir
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("read current branch: %w", err)
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("current checkout is detached; pass --head explicitly")
	}
	return branch, nil
}
