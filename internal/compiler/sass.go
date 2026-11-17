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
var sassVariableDeclaration = regexp.MustCompile(`(?m)^[\t ]*\$([a-zA-Z_][a-zA-Z0-9_-]*)[\t ]*:`)
var sassVariableReference = regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_-]*)`)
var fashionDynamicVariable = regexp.MustCompile(`^([\t ]*\$[a-zA-Z_][a-zA-Z0-9_-]*[\t ]*:[\t ]*)dynamic\([\t ]*(#[0-9a-fA-F]{3,8}|[-+]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[a-zA-Z%]+)?|[a-zA-Z_-][a-zA-Z0-9_-]*|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')[\t ]*\)([\t ]*;[\t ]*(?://.*)?)?$`)

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

func expandPackageClassSassRoots(pkg *sdkPackage, toolkit, locale string) ([]packageSassSourceRoot, []string, error) {
	paths, err := expandPackagePaths(pkg, pkg.manifest.Sencha.Sass.Src, toolkit, locale)
	if err != nil {
		return nil, nil, err
	}
	var roots []packageSassSourceRoot
	var warnings []string
	for _, path := range paths {
		sassRoot := filepath.Clean(path.path)
		sassSlash := filepath.ToSlash(sassRoot)
		if !strings.HasSuffix(sassSlash, "/sass/src") {
			warnings = append(warnings, fmt.Sprintf("skipped package Sass path %s: expected a package sass/src directory", sassRoot))
			continue
		}
		classRoot := filepath.FromSlash(strings.TrimSuffix(sassSlash, "/sass/src") + "/src")
		if !isPathWithin(sassRoot, pkg.root) || !isPathWithin(classRoot, pkg.root) {
			return nil, nil, fmt.Errorf("package %q Sass path escapes package directory", pkg.manifest.Sencha.Name)
		}
		roots = append(roots, packageSassSourceRoot{SassPath: sassRoot, ClassPath: classRoot, Package: pkg.manifest.Sencha.Name})
	}
	return roots, warnings, nil
}

func resolvePackageSassVariables(appRoot string, roots []string, initialVariables map[string]bool, ignore []*regexp.Regexp) ([]string, []string, map[string]bool, []string, error) {
	available := make(map[string]bool, len(initialVariables))
	for name := range initialVariables {
		available[name] = true
	}
	var sources []string
	var watchPaths []string
	var warnings []string
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, nil, nil, fmt.Errorf("stat package Sass var path %s: %w", root, err)
		}
		files := []string{root}
		if info.IsDir() {
			files = nil
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
				if strings.EqualFold(filepath.Ext(path), ".scss") && !isIgnoredProjectPath(appRoot, path, ignore) {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("scan package Sass var path %s: %w", root, err)
			}
		}
		for _, file := range files {
			if !strings.EqualFold(filepath.Ext(file), ".scss") || isIgnoredProjectPath(appRoot, file, ignore) {
				continue
			}
			absolute, err := filepath.Abs(file)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			watchPaths = append(watchPaths, absolute)
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("read package Sass var source %s: %w", file, err)
			}
			definitions := sassVariableNames(data)
			if len(definitions) == 0 {
				continue
			}
			if reason := unsupportedPackageSassVariableSource(data, definitions, available); reason != "" {
				warnings = append(warnings, fmt.Sprintf("skipped package Sass var source %s: %s", file, reason))
				continue
			}
			sources = append(sources, absolute)
			for name := range definitions {
				available[name] = true
			}
		}
	}
	return uniquePaths(sources), uniquePaths(watchPaths), available, warnings, nil
}

func collectSassVariableNames(sources []string) (map[string]bool, error) {
	variables := make(map[string]bool)
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read Sass source %s: %w", source, err)
		}
		if unsupportedFashionSource(data) != "" {
			continue
		}
		for name := range sassVariableNames(data) {
			variables[name] = true
		}
	}
	return variables, nil
}

func sassVariableNames(source []byte) map[string]bool {
	variables := make(map[string]bool)
	for _, match := range sassVariableDeclaration.FindAllSubmatch(source, -1) {
		variables[string(match[1])] = true
	}
	return variables
}

func unsupportedPackageSassVariableSource(source []byte, definitions, available map[string]bool) string {
	if reason := unsupportedFashionSource(source); reason != "" {
		return reason
	}
	for _, marker := range []string{"@import", "@use", "@forward", "url(", "#{"} {
		if bytes.Contains(source, []byte(marker)) {
			return "package Sass var uses imports, interpolation or resource URLs not covered by this step"
		}
	}
	for _, match := range sassVariableReference.FindAllSubmatch(source, -1) {
		name := string(match[1])
		if !definitions[name] && !available[name] {
			return fmt.Sprintf("references undefined package/theme variable $%s", name)
		}
	}
	return ""
}

func resolvePackageClassSassSources(appRoot string, roots []packageSassSourceRoot, variables map[string]bool, ignore []*regexp.Regexp) ([]string, []string, []string, error) {
	var sources []string
	var watchPaths []string
	var warnings []string
	for _, root := range roots {
		info, err := os.Stat(root.SassPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, nil, fmt.Errorf("stat Sass source path %s: %w", root.SassPath, err)
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(root.SassPath, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if path != root.SassPath && isIgnoredProjectPath(appRoot, path, ignore) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.EqualFold(filepath.Ext(path), ".scss") || isIgnoredProjectPath(appRoot, path, ignore) {
				return nil
			}
			relative, err := filepath.Rel(root.SassPath, path)
			if err != nil {
				return err
			}
			classPath := filepath.Join(root.ClassPath, strings.TrimSuffix(relative, filepath.Ext(relative))+".js")
			classInfo, err := os.Stat(classPath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return fmt.Errorf("stat package class source %s: %w", classPath, err)
			}
			if !classInfo.Mode().IsRegular() {
				return nil
			}
			absolute, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			watchPaths = append(watchPaths, absolute)
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read package Sass source %s: %w", path, err)
			}
			if reason := unsupportedPackageSassSource(data, variables); reason != "" {
				warnings = append(warnings, fmt.Sprintf("skipped package Sass source %s (%s): %s", path, root.Package, reason))
				return nil
			}
			sources = append(sources, absolute)
			return nil
		})
		if err != nil {
			return nil, nil, nil, fmt.Errorf("scan package Sass path %s: %w", root.SassPath, err)
		}
	}
	return uniquePaths(sources), uniquePaths(watchPaths), warnings, nil
}

func unsupportedPackageSassSource(source []byte, availableVariables map[string]bool) string {
	if reason := unsupportedFashionSource(source); reason != "" {
		return reason
	}
	if bytes.Contains(source, []byte("@import")) || bytes.Contains(source, []byte("@use")) || bytes.Contains(source, []byte("@forward")) {
		return "package Sass imports are not yet resolved"
	}
	if bytes.Contains(source, []byte("url(")) {
		return "package Sass resource URLs are not yet rebased"
	}
	definitions := sassVariableNames(source)
	for _, match := range sassVariableReference.FindAllSubmatch(source, -1) {
		name := string(match[1])
		if !definitions[name] && !availableVariables[name] {
			return fmt.Sprintf("references package/theme variable $%s not loaded by sass.var", name)
		}
	}
	return ""
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
		normalized, dynamicChanged, reason := normalizeFashionDynamic(data)
		if reason != "" {
			warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: %s", source, reason))
			continue
		}
		if reason := unsupportedFashionSource(normalized); reason != "" {
			warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: %s", source, reason))
			continue
		}
		if dynamicChanged {
			if len(sassImportDirective.FindAllSubmatch(normalized, -1)) > 0 {
				warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: dynamic() normalization with imports is not supported", source))
				continue
			}
			entry.Write(normalized)
			if len(normalized) > 0 && normalized[len(normalized)-1] != '\n' {
				entry.WriteByte('\n')
			}
			imported++
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

func normalizeFashionDynamic(source []byte) ([]byte, bool, string) {
	lines := strings.SplitAfter(string(source), "\n")
	var output strings.Builder
	changed := false
	for _, line := range lines {
		content := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		lineEnding := line[len(content):]
		trimmed := strings.TrimSpace(content)
		if !strings.Contains(content, "dynamic(") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			output.WriteString(line)
			continue
		}
		match := fashionDynamicVariable.FindStringSubmatch(content)
		if match == nil {
			return nil, false, "dynamic() requires Fashion dependency evaluation; only simple variable literals are supported"
		}
		output.WriteString(match[1])
		output.WriteString(match[2])
		output.WriteString(match[3])
		output.WriteString(lineEnding)
		changed = true
	}
	return []byte(output.String()), changed, ""
}

func unsupportedFashionSource(source []byte) string {
	if _, _, reason := normalizeFashionDynamic(source); reason != "" {
		return reason
	}
	for _, line := range strings.Split(string(source), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "@include") {
			return "Sass mixins are not loaded without the Sencha Fashion theme context"
		}
	}
	return ""
}
