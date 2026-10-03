package infrastructure

import (
	"context"
	"fmt"
	"strings"
)

const MaxRepositoryDiffCharacters = 40000

func ReadRepositoryDiff(
	ctx context.Context,
	repositoryPath string,
	baseRef string,
	includeUncommitted bool,
	filePath string,
) (string, bool, error) {
	arguments := []string{"diff", "--no-color"}

	if trimmedBaseRef := strings.TrimSpace(baseRef); trimmedBaseRef != "" {
		arguments = append(arguments, trimmedBaseRef)
	}
	if includeUncommitted {
		arguments = append(arguments, "HEAD")
	}
	if trimmedFilePath := strings.TrimSpace(filePath); trimmedFilePath != "" {
		arguments = append(arguments, "--", trimmedFilePath)
	}

	output, err := runGit(ctx, repositoryPath, arguments...)
	if err != nil {
		return "", false, fmt.Errorf("readRepositoryDiff: %w", err)
	}

	if len(output) > MaxRepositoryDiffCharacters {
		return output[:MaxRepositoryDiffCharacters], true, nil
	}
	return output, false, nil
}
