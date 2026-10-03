package application

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	sharedErrors "lunar/backend/internal/shared/errors"
	"lunar/backend/internal/workspace/domain"
)

func parseWorkspaceSource(rawSource string) (domain.WorkspaceSource, error) {
	switch domain.WorkspaceSource(strings.TrimSpace(rawSource)) {
	case domain.SourceHelper:
		return domain.SourceHelper, nil
	case domain.SourceServer:
		return domain.SourceServer, nil
	default:
		return "", fmt.Errorf(
			"%w: source must be %q or %q",
			sharedErrors.ErrBadRequest,
			domain.SourceHelper,
			domain.SourceServer,
		)
	}
}

func validateSelectionBatch(repoNames []string) ([]string, error) {
	seenNames := make(map[string]bool, len(repoNames))
	validatedNames := make([]string, 0, len(repoNames))

	for _, repoName := range repoNames {
		validName, err := domain.ValidateRepoName(repoName)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", sharedErrors.ErrBadRequest, err.Error())
		}
		if seenNames[validName] {
			continue
		}
		seenNames[validName] = true
		validatedNames = append(validatedNames, validName)
	}

	return validatedNames, nil
}

func validateHelperURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: helper url is not a valid url", sharedErrors.ErrBadRequest)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("%w: helper url must use http or https", sharedErrors.ErrBadRequest)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("%w: helper url must include a host", sharedErrors.ErrBadRequest)
	}

	return nil
}

func describeRootError(rootErr error) string {
	switch {
	case errors.Is(rootErr, domain.ErrRootNotFound):
		return "workspace root does not exist"
	case errors.Is(rootErr, domain.ErrRootNotDirectory):
		return "workspace root is not a directory"
	case errors.Is(rootErr, domain.ErrRootOutsideScope):
		return "workspace root is outside the allowed roots"
	case errors.Is(rootErr, domain.ErrRootNotAllowed):
		return "workspace root is not permitted"
	default:
		return "workspace root is invalid"
	}
}
