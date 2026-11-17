package compiler

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var sassImportDirective = regexp.MustCompile(`(?m)^[\t ]*@(import|use|forward)[\t ]+([^;\n]+)`)
var sassImportPath = regexp.MustCompile(`["']([^"']+)["']`)

func resolveClassSassSources(appRoot string, sourceRoots []string, ignore []*regexp.Regexp) ([]string, error) {
	var sources []string
	for _, sourceRoot := range sourceRoots {
		root, err := projectPath(appRoot, sourceRoot)
		if err != nil {
			return nil, fmt.Errorf("Sass source path: %w", err)
		}
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat Sass source path %s: %w", root, err)
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != root && isIgnoredProjectPath(appRoot, path, ignore) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.EqualFold(filepath.Ext(path), ".scss") || isIgnoredProjectPath(appRoot, path, ignore) {
				return nil
			}
			classSource := strings.TrimSuffix(path, filepath.Ext(path)) + ".js"
			classInfo, err := os.Stat(classSource)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return fmt.Errorf("stat class source for Sass file %s: %w", path, err)
			}
			if classInfo.Mode().IsRegular() {
				absolute, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				sources = append(sources, absolute)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan Sass source path %s: %w", root, err)
		}
	}
	return uniquePaths(sources), nil
}

func resolveConfiguredSassSources(appRoot, appName, toolkit string, config sassConfig, ignore []*regexp.Regexp) ([]string, error) {
	var sources []string
	for _, paths := range [][]string{config.Etc, config.Var, config.Src} {
		expanded, err := expandAppPaths(paths, appRoot, appName, toolkit)
		if err != nil {
			return nil, err
		}
		for _, configuredPath := range expanded {
			path, err := projectPath(appRoot, configuredPath)
			if err != nil {
				return nil, fmt.Errorf("configured Sass path: %w", err)
			}
			if isIgnoredProjectPath(appRoot, path, ignore) {
				continue
			}
			info, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("stat configured Sass path %s: %w", path, err)
			}
			if !info.IsDir() {
				if strings.EqualFold(filepath.Ext(path), ".scss") {
					absolute, err := filepath.Abs(path)
					if err != nil {
						return nil, err
					}
					sources = append(sources, absolute)
				}
				continue
			}
			err = filepath.WalkDir(path, func(source string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() {
					if source != path && isIgnoredProjectPath(appRoot, source, ignore) {
						return filepath.SkipDir
					}
					return nil
				}
				if strings.HasPrefix(entry.Name(), "_") || !strings.EqualFold(filepath.Ext(source), ".scss") || isIgnoredProjectPath(appRoot, source, ignore) {
					return nil
				}
				absolute, err := filepath.Abs(source)
				if err != nil {
					return err
				}
				sources = append(sources, absolute)
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("scan configured Sass path %s: %w", path, err)
			}
		}
	}
	return uniquePaths(sources), nil
}

func resolveSassImportWatchPaths(appRoot string, sources []string) ([]string, error) {
	seen := make(map[string]bool)
	visited := make(map[string]bool)
	var dependencies []string
	var visit func(string) error
	visit = func(source string) error {
		absolute, err := filepath.Abs(source)
		if err != nil {
			return err
		}
		if visited[absolute] {
			return nil
		}
		visited[absolute] = true
		data, err := os.ReadFile(absolute)
		if err != nil {
			return fmt.Errorf("read Sass import source %s: %w", absolute, err)
		}
		for _, importPath := range sassImportPaths(data) {
			dependency, exists, err := resolveSassImport(appRoot, absolute, importPath)
			if err != nil {
				return err
			}
			if !exists {
				continue
			}
			if !seen[dependency] {
				seen[dependency] = true
				dependencies = append(dependencies, dependency)
			}
			if err := visit(dependency); err != nil {
				return err
			}
		}
		return nil
	}
	for _, source := range sources {
		if err := visit(source); err != nil {
			return nil, err
		}
	}
	return dependencies, nil
}

func sassImportPaths(source []byte) []string {
	var paths []string
	for _, directive := range sassImportDirective.FindAllSubmatch(source, -1) {
		value := string(directive[2])
		if strings.Contains(value, "url(") {
			continue
		}
		for _, match := range sassImportPath.FindAllStringSubmatch(value, -1) {
			path := strings.TrimSpace(match[1])
			if path == "" || strings.HasPrefix(path, "sass:") || strings.Contains(path, "://") || strings.HasSuffix(strings.ToLower(path), ".css") {
				continue
			}
			paths = append(paths, path)
		}
	}
	return paths
}

func resolveSassImport(appRoot, importer, importPath string) (string, bool, error) {
	path := filepath.FromSlash(importPath)
	directories := []string{filepath.Dir(importer), appRoot}
	for _, directory := range directories {
		for _, candidate := range sassImportCandidates(filepath.Join(directory, path)) {
			absolute, err := filepath.Abs(candidate)
			if err != nil {
				return "", false, err
			}
			if !isPathWithin(absolute, appRoot) {
				continue
			}
			info, err := os.Stat(absolute)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return "", false, err
			}
			if info.Mode().IsRegular() && (strings.EqualFold(filepath.Ext(absolute), ".scss") || strings.EqualFold(filepath.Ext(absolute), ".sass")) {
				return absolute, true, nil
			}
		}
	}
	return "", false, nil
}

func sassImportCandidates(path string) []string {
	extension := strings.ToLower(filepath.Ext(path))
	if extension == ".scss" || extension == ".sass" {
		directory := filepath.Dir(path)
		base := filepath.Base(path)
		return []string{path, filepath.Join(directory, "_"+base)}
	}
	directory := filepath.Dir(path)
	base := filepath.Base(path)
	var candidates []string
	for _, extension := range []string{".scss", ".sass"} {
		candidates = append(candidates,
			path+extension,
			filepath.Join(directory, "_"+base+extension),
			filepath.Join(path, "index"+extension),
			filepath.Join(path, "_index"+extension),
		)
	}
	return candidates
}

func compileClassSass(appRoot string, sources []string, minify bool) ([]byte, []string, error) {
	if len(sources) == 0 {
		return nil, nil, nil
	}
	binary := strings.TrimSpace(os.Getenv("SASS"))
	if binary == "" {
		binary = "sass"
	}
	style := "expanded"
	if minify {
		style = "compressed"
	}
	var entry strings.Builder
	var warnings []string
	imported := 0
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, warnings, fmt.Errorf("read Sass source %s: %w", source, err)
		}
		if reason := unsupportedFashionSource(data); reason != "" {
			warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: %s", source, reason))
			continue
		}
		relative, err := filepath.Rel(appRoot, source)
		if err != nil {
			return nil, warnings, fmt.Errorf("make Sass source path relative: %w", err)
		}
		entry.WriteString("@import ")
		entry.WriteString(strconv.Quote(filepath.ToSlash(relative)))
		entry.WriteString(";\n")
		imported++
	}
	if imported == 0 {
		return nil, warnings, nil
	}
	command := exec.Command(binary, "--no-source-map", "--style="+style, "--load-path", appRoot, "--stdin")
	command.Stdin = strings.NewReader(entry.String())
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if os.IsNotExist(err) {
			return nil, warnings, fmt.Errorf("Dart Sass executable %q was not found; install Dart Sass or set SASS to its path", binary)
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, warnings, fmt.Errorf("compile class Sass with %s: %s", binary, detail)
	}
	if bytes.Contains(stdout.Bytes(), []byte("dynamic(")) {
		return nil, warnings, fmt.Errorf("compiled CSS still contains dynamic(); Sencha Fashion is required for this Sass input")
	}
	return stdout.Bytes(), warnings, nil
}

func unsupportedFashionSource(source []byte) string {
	if bytes.Contains(source, []byte("dynamic(")) {
		return "dynamic() requires Sencha Fashion"
	}
	for _, line := range strings.Split(string(source), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "@include") {
			return "Sass mixins are not loaded without the Sencha Fashion theme context"
		}
	}
	return ""
}
