package compiler

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

func compileIgnorePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		matcher, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid app.json ignore pattern %q: %w", pattern, err)
		}
		compiled = append(compiled, matcher)
	}
	return compiled, nil
}

func isIgnoredProjectPath(appRoot, sourcePath string, patterns []*regexp.Regexp) bool {
	if len(patterns) == 0 {
		return false
	}
	relative, err := filepath.Rel(appRoot, sourcePath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return false
	}
	name := filepath.ToSlash(relative)
	for _, pattern := range patterns {
		if pattern.MatchString(name) {
			return true
		}
	}
	return false
}
