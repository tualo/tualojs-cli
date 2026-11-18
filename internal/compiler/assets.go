package compiler

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type resolvedResource struct {
	source string
	output string
}

type resolvedStylesheet struct {
	source string
	url    string
	remote bool
}

type resolvedScript struct {
	asset           ScriptAsset
	source          string
	output          string
	includeInBundle bool
}

type StyleSheet struct {
	URL    string
	Path   string
	Remote bool
}

func resolveStylesheets(appRoot string, config appConfig, toolkit, mode string, ignore []*regexp.Regexp) ([]resolvedStylesheet, []string, error) {
	entries := append(resourceList{}, config.CSS...)
	if toolkit == "classic" {
		entries = append(entries, config.Classic.CSS...)
	} else if toolkit == "modern" {
		entries = append(entries, config.Modern.CSS...)
	}
	var stylesheets []resolvedStylesheet
	var watchPaths []string
	for _, entry := range entries {
		if entry.Path == "" {
			continue
		}
		path := strings.ReplaceAll(entry.Path, "${build.id}", resolveBuildID(appRoot, config.Name, mode))
		paths, err := expandAppPaths([]string{path}, appRoot, config.Name, toolkit)
		if err != nil {
			return nil, nil, err
		}
		path = paths[0]
		if strings.Contains(path, "${") {
			continue
		}
		if entry.Remote {
			stylesheets = append(stylesheets, resolvedStylesheet{url: path, remote: true})
			continue
		}
		source, err := projectPath(appRoot, path)
		if err != nil {
			return nil, nil, fmt.Errorf("stylesheet source: %w", err)
		}
		if isIgnoredProjectPath(appRoot, source, ignore) {
			continue
		}
		info, err := os.Stat(source)
		if err != nil {
			if strings.Contains(entry.Path, "${build.id}") && os.IsNotExist(err) {
				continue
			}
			return nil, nil, fmt.Errorf("stylesheet source %q: %w", entry.Path, err)
		}
		if !info.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("stylesheet source %s is not a regular file", source)
		}
		relative, err := filepath.Rel(appRoot, source)
		if err != nil {
			return nil, nil, err
		}
		url := filepath.ToSlash(relative)
		stylesheets = append(stylesheets, resolvedStylesheet{source: source, url: url})
		watchPaths = append(watchPaths, source)
	}
	return stylesheets, uniquePaths(watchPaths), nil
}

func copyStylesheets(stylesheets []resolvedStylesheet, outputRoot, bundlePath string) ([]StyleSheet, error) {
	var copied []StyleSheet
	written := make(map[string]string)
	for _, stylesheet := range stylesheets {
		if stylesheet.remote {
			copied = append(copied, StyleSheet{URL: stylesheet.url, Remote: true})
			continue
		}
		destination := filepath.Join(outputRoot, filepath.FromSlash(stylesheet.url))
		absolute, err := filepath.Abs(destination)
		if err != nil {
			return nil, err
		}
		if absolute == bundlePath {
			return nil, fmt.Errorf("stylesheet output would overwrite JavaScript bundle %s", bundlePath)
		}
		if err := copyResourceFile(stylesheet.source, absolute, written, false); err != nil {
			return nil, fmt.Errorf("copy stylesheet: %w", err)
		}
		copied = append(copied, StyleSheet{URL: stylesheet.url, Path: absolute})
	}
	return copied, nil
}

func resolveScripts(appRoot, appName string, entries []jsEntry, toolkit string, ignore []*regexp.Regexp) ([]resolvedScript, []string, error) {
	var scripts []resolvedScript
	var watchPaths []string
	for _, entry := range entries {
		if entry.Bundle {
			scripts = append(scripts, resolvedScript{asset: ScriptAsset{URL: "app.js"}})
			continue
		}
		if entry.IncludeInBundle && entry.Remote {
			return nil, nil, fmt.Errorf("JavaScript asset %q cannot be both remote and includeInBundle", entry.Path)
		}
		if entry.Remote {
			scripts = append(scripts, resolvedScript{asset: ScriptAsset{URL: entry.Path, Remote: true}})
			continue
		}
		if strings.Contains(entry.Path, "${framework.dir}") || strings.Contains(entry.Path, "${build.") {
			continue
		}
		paths, err := expandAppPaths([]string{entry.Path}, appRoot, appName, toolkit)
		if err != nil {
			return nil, nil, err
		}
		source, err := projectPath(appRoot, paths[0])
		if err != nil {
			return nil, nil, fmt.Errorf("JavaScript asset: %w", err)
		}
		if isIgnoredProjectPath(appRoot, source, ignore) {
			continue
		}
		info, err := os.Stat(source)
		if err != nil {
			return nil, nil, fmt.Errorf("JavaScript asset %q: %w", entry.Path, err)
		}
		if !info.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("JavaScript asset %s is not a regular file", source)
		}
		relative, err := filepath.Rel(appRoot, source)
		if err != nil {
			return nil, nil, err
		}
		url := filepath.ToSlash(filepath.Join("js", relative))
		if entry.IncludeInBundle {
			scripts = append(scripts, resolvedScript{source: source, output: filepath.ToSlash(relative), includeInBundle: true})
			watchPaths = append(watchPaths, source)
			continue
		}
		scripts = append(scripts, resolvedScript{
			asset:  ScriptAsset{URL: url},
			source: source,
			output: url,
		})
		watchPaths = append(watchPaths, source)
	}
	return scripts, uniquePaths(watchPaths), nil
}

func copyScripts(scripts []resolvedScript, outputRoot, bundlePath string) ([]ScriptAsset, error) {
	assets := make([]ScriptAsset, 0, len(scripts))
	for _, script := range scripts {
		if script.includeInBundle {
			continue
		}
		if script.source == "" {
			assets = append(assets, script.asset)
			continue
		}
		destination := filepath.Join(outputRoot, filepath.FromSlash(script.output))
		absolute, err := filepath.Abs(destination)
		if err != nil {
			return nil, err
		}
		if absolute == bundlePath {
			return nil, fmt.Errorf("JavaScript asset would overwrite bundle %s", bundlePath)
		}
		data, err := os.ReadFile(script.source)
		if err != nil {
			return nil, fmt.Errorf("read JavaScript asset %s: %w", script.source, err)
		}
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			return nil, fmt.Errorf("create JavaScript asset directory: %w", err)
		}
		if err := writeFileAtomic(absolute, data); err != nil {
			return nil, fmt.Errorf("write JavaScript asset %s: %w", absolute, err)
		}
		assets = append(assets, script.asset)
	}
	return assets, nil
}

func resolveResources(appRoot, appName, mode string, entries resourceList, toolkit string, ignore []*regexp.Regexp) ([]resolvedResource, []string, error) {
	var resources []resolvedResource
	var watchPaths []string
	for _, entry := range entries {
		if entry.Remote {
			continue
		}
		if entry.Path == "" {
			return nil, nil, fmt.Errorf("app.json contains a resource entry with an empty path")
		}
		buildID := resolveBuildID(appRoot, appName, mode)
		resourcePath := strings.ReplaceAll(entry.Path, "${build.id}", buildID)
		paths, err := expandAppPaths([]string{resourcePath}, appRoot, appName, toolkit)
		if err != nil {
			return nil, nil, err
		}
		source, err := projectPath(appRoot, paths[0])
		if err != nil {
			return nil, nil, fmt.Errorf("resource source: %w", err)
		}
		if isIgnoredProjectPath(appRoot, source, ignore) {
			continue
		}
		info, err := os.Stat(source)
		if err != nil {
			if strings.Contains(entry.Path, "${build.id}") && os.IsNotExist(err) {
				continue
			}
			return nil, nil, fmt.Errorf("resource source %q: %w", entry.Path, err)
		}
		output := strings.ReplaceAll(entry.Output, "${build.id}", buildID)
		if output != "" {
			outputs, err := expandAppPaths([]string{output}, appRoot, appName, toolkit)
			if err != nil {
				return nil, nil, err
			}
			output = outputs[0]
		} else if info.IsDir() && filepath.Base(filepath.Clean(paths[0])) == "resources" {
			if toolkit != "" && (strings.Contains(entry.Path, "${toolkit.name}") || filepath.ToSlash(filepath.Clean(paths[0])) == filepath.ToSlash(filepath.Join(toolkit, "resources"))) {
				output = filepath.ToSlash(filepath.Join(toolkit, "resources"))
			} else {
				output = "."
			}
		} else {
			relative, err := filepath.Rel(appRoot, source)
			if err != nil {
				return nil, nil, err
			}
			output = filepath.ToSlash(filepath.Clean(relative))
		}
		if err := validateResourceOutput(output); err != nil {
			return nil, nil, fmt.Errorf("resource output for %q: %w", entry.Path, err)
		}
		resources = append(resources, resolvedResource{source: source, output: output})
		watchPaths = append(watchPaths, source)
		if toolkit != "" && strings.Contains(entry.Path, "${build.id}") {
			toolkitResources := filepath.Join(appRoot, filepath.FromSlash(buildID), toolkit, "resources")
			if toolkitInfo, statErr := os.Stat(toolkitResources); statErr == nil && toolkitInfo.IsDir() {
				if !isIgnoredProjectPath(appRoot, toolkitResources, ignore) {
					resources = append(resources, resolvedResource{source: toolkitResources, output: "shared"})
					watchPaths = append(watchPaths, toolkitResources)
				}
			} else if statErr != nil && !os.IsNotExist(statErr) {
				return nil, nil, fmt.Errorf("inspect toolkit build resources %s: %w", toolkitResources, statErr)
			}
		}
	}
	return resources, uniquePaths(watchPaths), nil
}

func resolveBuildID(appRoot, appName, mode string) string {
	order := []string{mode, "production", "development"}
	seen := make(map[string]bool)
	for _, candidateMode := range order {
		if candidateMode == "" || seen[candidateMode] {
			continue
		}
		seen[candidateMode] = true
		candidate := filepath.Join("build", candidateMode, appName)
		if info, err := os.Stat(filepath.Join(appRoot, candidate)); err == nil && info.IsDir() {
			return filepath.ToSlash(candidate)
		}
	}
	return filepath.ToSlash(filepath.Join("build", mode, appName))
}

func validateResourceOutput(output string) error {
	if filepath.IsAbs(output) {
		return fmt.Errorf("absolute output path is not allowed")
	}
	clean := filepath.Clean(filepath.FromSlash(output))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("output path escapes resources directory")
	}
	return nil
}

func copyResources(resources []resolvedResource, outputRoot, appRoot string, ignore []*regexp.Regexp) (int, error) {
	resourceRoot := filepath.Join(outputRoot, "resources")
	written := make(map[string]string)
	for _, resource := range resources {
		info, err := os.Stat(resource.source)
		if err != nil {
			return len(written), fmt.Errorf("stat resource %s: %w", resource.source, err)
		}
		if info.IsDir() {
			targetBase := filepath.Join(resourceRoot, filepath.FromSlash(resource.output))
			if isPathWithin(targetBase, resource.source) {
				return len(written), fmt.Errorf("resource output %s is inside its source directory %s", targetBase, resource.source)
			}
			err = filepath.WalkDir(resource.source, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.Type()&os.ModeSymlink != 0 {
					if entry.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if isIgnoredProjectPath(appRoot, sourcePath, ignore) {
					if entry.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if entry.IsDir() {
					return nil
				}
				relative, err := filepath.Rel(resource.source, sourcePath)
				if err != nil {
					return err
				}
				destination := filepath.Join(targetBase, relative)
				if err := copyResourceFile(sourcePath, destination, written, true); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return len(written), fmt.Errorf("copy resource directory %s: %w", resource.source, err)
			}
			continue
		}
		destination := filepath.Join(resourceRoot, filepath.FromSlash(resource.output))
		if resource.output == "." || filepath.Ext(resource.output) == "" {
			destination = filepath.Join(destination, filepath.Base(resource.source))
		}
		if err := copyResourceFile(resource.source, destination, written, true); err != nil {
			return len(written), err
		}
	}
	return len(written), nil
}

func copyResourceFile(source, destination string, written map[string]string, overwrite bool) error {
	destination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if previous, exists := written[destination]; exists && previous != source {
		if !overwrite {
			return fmt.Errorf("resource collision: %s and %s both map to %s", previous, source, destination)
		}
	} else if exists {
		return nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read resource %s: %w", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create resource directory: %w", err)
	}
	if err := writeFileAtomic(destination, data); err != nil {
		return fmt.Errorf("write resource %s: %w", destination, err)
	}
	written[destination] = source
	return nil
}

func isPathWithin(path, parent string) bool {
	path, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(parent, path)
	return err == nil && (relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
