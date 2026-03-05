package finder

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	adapterhelpers "personalagent/runtime/internal/connectors/adapters/helpers"
)

var (
	finderQueryTokenRegex = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9._-]*`)
	errFinderNoMatches    = errors.New("finder query returned no matches")
	errFinderAmbiguous    = errors.New("finder query returned multiple matches")
)

type finderStepInput struct {
	Path     string
	Query    string
	RootPath string
}

type finderTargetResolution struct {
	ResolvedPath string
	Query        string
	RootPath     string
	Candidates   []finderSearchCandidate
	ResolvedBy   string
}

type finderTargetError struct {
	Reason  string
	Summary string
	Err     error
}

func (e finderTargetError) Error() string {
	if strings.TrimSpace(e.Summary) != "" {
		return e.Summary
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "finder target resolution failed"
}

func (e finderTargetError) Unwrap() error {
	return e.Err
}

func resolveFinderInput(input map[string]any) (finderStepInput, error) {
	if len(input) == 0 {
		return finderStepInput{}, fmt.Errorf("finder step input is required")
	}
	pathValue, err := adapterhelpers.OptionalStringInput(input, "path")
	if err != nil {
		return finderStepInput{}, err
	}
	queryValue, err := adapterhelpers.OptionalStringInput(input, "query")
	if err != nil {
		return finderStepInput{}, err
	}
	rootPathValue, err := adapterhelpers.OptionalStringInput(input, "root_path")
	if err != nil {
		return finderStepInput{}, err
	}
	if pathValue == "" && queryValue == "" {
		return finderStepInput{}, fmt.Errorf("finder step input requires path or query")
	}
	return finderStepInput{
		Path:     pathValue,
		Query:    queryValue,
		RootPath: rootPathValue,
	}, nil
}

func resolveFinderTarget(capability string, input map[string]any) (finderTargetResolution, error) {
	finderInput, err := resolveFinderInput(input)
	if err != nil {
		return finderTargetResolution{}, finderTargetError{
			Reason:  "invalid_path",
			Summary: "finder target path or query is required",
			Err:     err,
		}
	}

	if finderInput.Path != "" {
		targetPath := filepath.Clean(finderInput.Path)
		if !filepath.IsAbs(targetPath) {
			return finderTargetResolution{}, finderTargetError{
				Reason:  "invalid_path",
				Summary: "finder path must be absolute",
				Err:     fmt.Errorf("finder path must be absolute: %s", finderInput.Path),
			}
		}
		return finderTargetResolution{
			ResolvedPath: targetPath,
			ResolvedBy:   "path",
		}, nil
	}

	if strings.TrimSpace(finderInput.Query) == "" {
		return finderTargetResolution{}, finderTargetError{
			Reason:  "invalid_query",
			Summary: "finder query is required when path is omitted",
			Err:     fmt.Errorf("finder query is required when path is omitted"),
		}
	}
	candidates, resolvedRoot, searchErr := semanticFindCandidates(finderInput.Query, finderInput.RootPath)
	if searchErr != nil {
		return finderTargetResolution{}, finderTargetError{
			Reason:  "search_failed",
			Summary: "finder query resolution failed",
			Err:     searchErr,
		}
	}
	if len(candidates) == 0 {
		return finderTargetResolution{}, finderTargetError{
			Reason:  "not_found",
			Summary: "finder query returned no matches",
			Err:     errFinderNoMatches,
		}
	}
	if capability == CapabilityDelete && len(candidates) > 1 {
		return finderTargetResolution{}, finderTargetError{
			Reason:  "guardrail_denied",
			Summary: "finder delete query is ambiguous; provide an absolute path or more specific query",
			Err:     errFinderAmbiguous,
		}
	}
	return finderTargetResolution{
		ResolvedPath: candidates[0].Path,
		Query:        finderInput.Query,
		RootPath:     resolvedRoot,
		Candidates:   candidates,
		ResolvedBy:   "query",
	}, nil
}
