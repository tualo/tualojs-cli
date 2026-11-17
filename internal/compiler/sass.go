package compiler

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var sassImportDirective = regexp.MustCompile(`(?m)^[\t ]*@(import|use|forward)[\t ]+([^;\n]+)`)
var sassImportPath = regexp.MustCompile(`["']([^"']+)["']`)
var sassVariableDeclaration = regexp.MustCompile(`(?m)^[\t ]*\$([a-zA-Z_][a-zA-Z0-9_-]*)[\t ]*:`)
var sassVariableReference = regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_-]*)`)
var sassMixinDefinition = regexp.MustCompile(`(?m)^[\t ]*@mixin[\t ]+([a-zA-Z_-][a-zA-Z0-9_-]*)`)
var sassMixinInclude = regexp.MustCompile(`@include[\t ]+([a-zA-Z_-][a-zA-Z0-9_-]*)`)
var sassFunctionDefinition = regexp.MustCompile(`(?m)^[\t ]*@function[\t ]+([a-zA-Z_-][a-zA-Z0-9_-]*)`)
var fashionBoxFunctionArgs = regexp.MustCompile(`(?m)(@function[\t ]+(?:top|right|bottom|left|vertical|horizontal|boxmax|boxmin)[\t ]*\([\t ]*\$box[\t ]*,[\t ]*\$fallbackBox)([\t ]*\))`)
var fashionDynamicAssignment = regexp.MustCompile(`^[\t ]*\$[a-zA-Z_][a-zA-Z0-9_-]*[\t ]*:[\t ]*dynamic\(`)
var fashionDynamicSuffix = regexp.MustCompile(`^[\t ]*(?:!default[\t ]*)?;?[\t ]*(?://.*)?$`)

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

func uniquePackageSassSourceRoots(roots []packageSassSourceRoot) []packageSassSourceRoot {
	seen := make(map[string]bool, len(roots))
	unique := make([]packageSassSourceRoot, 0, len(roots))
	for _, root := range roots {
		key := filepath.Clean(root.SassPath) + "\x00" + filepath.Clean(root.ClassPath)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, root)
	}
	return unique
}

func resolvePackageSassEtcSources(appRoot string, roots []packageSassEtcRoot, ignore []*regexp.Regexp) ([]string, []string, []string, []string, error) {
	var sources []string
	var themeSources []string
	var watchPaths []string
	var warnings []string
	seen := make(map[string]bool)
	addSource := func(path string, theme bool) error {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if seen[absolute] {
			return nil
		}
		seen[absolute] = true
		watchPaths = append(watchPaths, absolute)
		if theme {
			themeSources = append(themeSources, absolute)
		} else {
			sources = append(sources, absolute)
		}
		return nil
	}
	for _, packageRoot := range roots {
		root := packageRoot.Path
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, nil, nil, fmt.Errorf("stat package Sass etc path %s: %w", root, err)
		}
		if !info.IsDir() {
			if strings.EqualFold(filepath.Ext(root), ".scss") && !isIgnoredProjectPath(appRoot, root, ignore) {
				if err := addSource(root, packageRoot.Theme); err != nil {
					return nil, nil, nil, nil, fmt.Errorf("resolve package Sass etc source %s: %w", root, err)
				}
			}
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
			if strings.HasPrefix(entry.Name(), "_") || !strings.EqualFold(filepath.Ext(path), ".scss") || isIgnoredProjectPath(appRoot, path, ignore) {
				return nil
			}
			return addSource(path, packageRoot.Theme)
		})
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("scan package Sass etc path %s: %w", root, err)
		}
	}
	return uniquePaths(sources), uniquePaths(themeSources), uniquePaths(watchPaths), warnings, nil
}

func resolveThemeSassHelperSources(appRoot string, roots []string) ([]string, error) {
	visited := make(map[string]bool)
	var helpers []string
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
			return fmt.Errorf("read theme Sass source %s: %w", absolute, err)
		}
		if sassSourceIsSassHelper(data) {
			helpers = append(helpers, absolute)
		}
		for _, importPath := range sassImportPaths(data) {
			dependency, exists, err := resolveSassImport(appRoot, absolute, importPath)
			if err != nil {
				return fmt.Errorf("resolve theme Sass import %q in %s: %w", importPath, absolute, err)
			}
			if exists {
				if err := visit(dependency); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, root := range roots {
		if err := visit(root); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
	}
	return uniquePaths(helpers), nil
}

func sassSourceIsSassHelper(source []byte) bool {
	if sassSourceHasNestedMixin(source) || (!sassFunctionDefinition.Match(source) && !sassMixinDefinition.Match(source)) {
		return false
	}
	definitions := sassVariableNames(source)
	depth := 0
	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if depth == 0 && trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") && !strings.HasPrefix(trimmed, "*") {
			if sassVariableDeclaration.MatchString(line) {
				for _, match := range sassVariableReference.FindAllSubmatch([]byte(line), -1) {
					name := string(match[1])
					if !definitions[name] {
						return false
					}
				}
			} else if strings.HasPrefix(trimmed, ".") || strings.HasPrefix(trimmed, "#") {
				return false
			}
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth < 0 {
			depth = 0
		}
	}
	return true
}

func normalizeFashionBoxFunctionDefaults(source []byte) []byte {
	return fashionBoxFunctionArgs.ReplaceAll(source, []byte("${1}: null${2}"))
}

func resolvePackageSassVariables(appRoot string, roots []string, initialVariables map[string]bool, ignore []*regexp.Regexp) ([]string, []string, map[string]bool, []string, error) {
	available := make(map[string]bool, len(initialVariables))
	for name := range initialVariables {
		available[name] = true
	}
	nullVariables := make(map[string]bool)
	var sources []string
	var watchPaths []string
	var warnings []string
	type variableSource struct {
		path        string
		data        []byte
		definitions map[string]bool
	}
	var pending []variableSource
	seenFiles := make(map[string]bool)
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
			if seenFiles[absolute] {
				continue
			}
			seenFiles[absolute] = true
			watchPaths = append(watchPaths, absolute)
			data, err := os.ReadFile(file)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("read package Sass var source %s: %w", file, err)
			}
			definitions := sassVariableNames(data)
			if len(definitions) == 0 {
				continue
			}
			pending = append(pending, variableSource{path: absolute, data: data, definitions: definitions})
		}
	}
	for len(pending) > 0 {
		remaining := make([]variableSource, 0, len(pending))
		progress := false
		for _, source := range pending {
			sourceNulls := sassVariableNullAssignments(source.data)
			candidateNulls := make(map[string]bool, len(nullVariables)+len(sourceNulls))
			for name, isNull := range nullVariables {
				candidateNulls[name] = isNull
			}
			for name, isNull := range sourceNulls {
				candidateNulls[name] = isNull
			}
			if name := sassVariableNullReference(source.data, candidateNulls); name != "" {
				progress = true
				warnings = append(warnings, fmt.Sprintf("skipped package Sass var source %s: references $%s whose Fashion value is null", source.path, name))
				continue
			}
			reason := unsupportedPackageSassVariableSource(source.data, source.definitions, available)
			if strings.HasPrefix(reason, "references undefined package/theme variable $") {
				remaining = append(remaining, source)
				continue
			}
			progress = true
			if reason != "" {
				warnings = append(warnings, fmt.Sprintf("skipped package Sass var source %s: %s", source.path, reason))
				continue
			}
			sources = append(sources, source.path)
			for name := range source.definitions {
				available[name] = true
			}
			for name, isNull := range sourceNulls {
				nullVariables[name] = isNull
			}
		}
		if !progress {
			for _, source := range remaining {
				reason := unsupportedPackageSassVariableSource(source.data, source.definitions, available)
				warnings = append(warnings, fmt.Sprintf("skipped package Sass var source %s: %s", source.path, reason))
			}
			break
		}
		pending = remaining
	}
	return uniquePaths(sources), uniquePaths(watchPaths), available, warnings, nil
}

func sassVariableNullAssignments(source []byte) map[string]bool {
	assignments := make(map[string]bool)
	if normalized, _, reason := normalizeFashionDynamic(source); reason == "" {
		source = normalized
	}
	for _, line := range strings.Split(string(source), "\n") {
		declaration := sassVariableDeclaration.FindStringSubmatch(line)
		if declaration == nil {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		value := strings.TrimSpace(line[colon+1:])
		if comment := strings.Index(value, "//"); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}
		value = strings.TrimSpace(strings.TrimSuffix(value, ";"))
		assignments[string(declaration[1])] = value == "null"
	}
	return assignments
}

func sassVariableNullReference(source []byte, nullVariables map[string]bool) string {
	for _, line := range strings.Split(string(source), "\n") {
		if !sassVariableDeclaration.MatchString(line) {
			continue
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			continue
		}
		for _, match := range sassVariableReference.FindAllSubmatch([]byte(line[colon+1:]), -1) {
			name := string(match[1])
			if nullVariables[name] {
				return name
			}
		}
	}
	return ""
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
	if len(sassMixinInclude.FindAllSubmatch(source, -1)) > 0 {
		return "Package Sass var mixin includes are not supported"
	}
	for _, marker := range []string{"@import", "@use", "@forward", "url("} {
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

func collectSassMixinNames(sources []string) (map[string]bool, error) {
	names := make(map[string]bool)
	for _, source := range uniquePaths(sources) {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read Sass mixin source %s: %w", source, err)
		}
		for _, match := range sassMixinDefinition.FindAllSubmatch(data, -1) {
			names[string(match[1])] = true
		}
	}
	return names, nil
}

func filterSassSourcesByMixins(sources []string, mixins map[string]bool) ([]string, []string, error) {
	var included []string
	var warnings []string
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, nil, fmt.Errorf("read Sass source %s: %w", source, err)
		}
		var missing []string
		for _, match := range sassMixinInclude.FindAllSubmatch(data, -1) {
			name := string(match[1])
			if !mixins[name] {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: mixins not found in loaded Sass sources: %s", source, strings.Join(uniqueStrings(missing), ", ")))
			continue
		}
		included = append(included, source)
	}
	return included, warnings, nil
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
			if path == "" || strings.HasPrefix(path, "sass:") || strings.Contains(path, "://") || strings.HasSuffix(strings.ToLower(path), ".css") || strings.HasSuffix(strings.ToLower(path), ".js") {
				continue
			}
			paths = append(paths, path)
		}
	}
	return paths
}

func inspectSassImportTree(appRoot, source string, visited map[string]bool) (bool, []string, error) {
	absolute, err := filepath.Abs(source)
	if err != nil {
		return false, nil, err
	}
	if visited[absolute] {
		return false, nil, nil
	}
	visited[absolute] = true
	data, err := os.ReadFile(absolute)
	if err != nil {
		return false, nil, fmt.Errorf("read Sass import source %s: %w", absolute, err)
	}
	var imports []string
	for _, directive := range sassImportDirective.FindAllSubmatch(data, -1) {
		value := string(directive[2])
		if strings.Contains(value, "url(") {
			continue
		}
		for _, match := range sassImportPath.FindAllStringSubmatch(value, -1) {
			importPath := strings.TrimSpace(match[1])
			if strings.HasSuffix(strings.ToLower(importPath), ".js") {
				return true, nil, nil
			}
			if importPath == "" || strings.HasPrefix(importPath, "sass:") || strings.Contains(importPath, "://") || strings.HasSuffix(strings.ToLower(importPath), ".css") {
				continue
			}
			dependency, exists, err := resolveSassImport(appRoot, absolute, importPath)
			if err != nil {
				return false, nil, err
			}
			if !exists {
				return true, uniquePaths(imports), nil
			}
			unsupported, nested, err := inspectSassImportTree(appRoot, dependency, visited)
			if err != nil {
				return false, nil, err
			}
			nestedImports := append([]string{dependency}, nested...)
			if unsupported {
				return true, uniquePaths(append(imports, nestedImports...)), nil
			}
			imports = append(imports, nestedImports...)
		}
	}
	return false, uniquePaths(imports), nil
}

func inlineSassImports(appRoot, importer string, source []byte, availableVariables map[string]bool, skipImports map[string]bool) ([]byte, []string, error) {
	var warnings []string
	active := make(map[string]bool)
	var expand func(string, []byte) ([]byte, error)
	expand = func(current string, data []byte) ([]byte, error) {
		data = normalizeFashionBoxFunctionDefaults(data)
		text := string(data)
		matches := sassImportDirective.FindAllStringSubmatchIndex(text, -1)
		if len(matches) == 0 {
			return data, nil
		}
		var output strings.Builder
		previous := 0
		braceDepth := 0
		for _, match := range matches {
			prefix := text[previous:match[0]]
			output.WriteString(prefix)
			braceDepth += strings.Count(prefix, "{") - strings.Count(prefix, "}")
			if braceDepth > 0 {
				warnings = append(warnings, fmt.Sprintf("skipped Sass import nested in a block in %s", current))
				previous = match[1]
				continue
			}
			directive := text[match[0]:match[1]]
			kind := text[match[2]:match[3]]
			value := text[match[4]:match[5]]
			if strings.Contains(value, "url(") {
				output.WriteString(directive)
				previous = match[1]
				continue
			}
			paths := sassImportPath.FindAllStringSubmatchIndex(value, -1)
			if len(paths) == 0 {
				output.WriteString(directive)
				previous = match[1]
				continue
			}
			var replacement strings.Builder
			valuePrevious := 0
			dropDirective := false
			inlinedImport := false
			for _, pathMatch := range paths {
				replacement.WriteString(value[valuePrevious:pathMatch[0]])
				literal := value[pathMatch[0]:pathMatch[1]]
				path := strings.TrimSpace(value[pathMatch[2]:pathMatch[3]])
				quote := literal[:1]
				if path == "" || strings.HasPrefix(path, "sass:") || strings.Contains(path, "://") || strings.HasSuffix(strings.ToLower(path), ".css") || filepath.IsAbs(filepath.FromSlash(path)) {
					replacement.WriteString(literal)
					valuePrevious = pathMatch[1]
					continue
				}
				if strings.HasSuffix(strings.ToLower(path), ".js") {
					warnings = append(warnings, fmt.Sprintf("skipped Fashion JavaScript import %q in Sass source %s", path, current))
					dropDirective = true
					valuePrevious = pathMatch[1]
					continue
				}
				resolved, exists, err := resolveSassImport(appRoot, current, path)
				if err != nil {
					return nil, fmt.Errorf("resolve Sass import %q in %s: %w", path, current, err)
				}
				if !exists {
					warnings = append(warnings, fmt.Sprintf("skipped unresolved Sass import %q in %s", path, current))
					dropDirective = true
					valuePrevious = pathMatch[1]
					continue
				}
				if kind == "import" {
					if skipImports[resolved] {
						inlinedImport = true
						valuePrevious = pathMatch[1]
						continue
					}
					if strings.Contains(strings.ToLower(filepath.Base(resolved)), "font-icons") {
						if missingVariable := sassImportTreeUnresolvedVariable([]string{resolved}, availableVariables); missingVariable != "" {
							warnings = append(warnings, fmt.Sprintf("skipped unsupported Sass import %q in %s: references unavailable Sass variable $%s", path, current, missingVariable))
							dropDirective = true
							valuePrevious = pathMatch[1]
							continue
						}
					}
					if active[resolved] {
						replacement.WriteString(literal)
						valuePrevious = pathMatch[1]
						continue
					}
					imported, err := os.ReadFile(resolved)
					if err != nil {
						return nil, fmt.Errorf("read Sass import source %s: %w", resolved, err)
					}
					if normalized, changed, reason := normalizeFashionDynamic(imported); reason == "" && changed {
						imported = normalized
					}
					if sassSourceHasNestedMixin(imported) {
						warnings = append(warnings, fmt.Sprintf("skipped Sass import %q in %s: nested Sass mixins are unsupported", path, current))
						dropDirective = true
						valuePrevious = pathMatch[1]
						continue
					}
					active[resolved] = true
					expanded, err := expand(resolved, imported)
					delete(active, resolved)
					if err != nil {
						return nil, err
					}
					if sassSourceNeedsTrailingSemicolon(expanded) {
						expanded = append(expanded, []byte("\n;\n")...)
					}
					replacement.Write(expanded)
					replacement.WriteByte('\n')
					inlinedImport = true
				} else {
					unsupported, _, err := inspectSassImportTree(appRoot, resolved, make(map[string]bool))
					if err != nil {
						return nil, err
					}
					if unsupported {
						warnings = append(warnings, fmt.Sprintf("skipped unsupported Sass import %q in %s", path, current))
						dropDirective = true
						valuePrevious = pathMatch[1]
						continue
					}
					replacement.WriteString(quote + filepath.ToSlash(resolved) + quote)
				}
				valuePrevious = pathMatch[1]
			}
			if !dropDirective {
				replacement.WriteString(value[valuePrevious:])
				if kind == "import" && inlinedImport {
					output.WriteString(replacement.String())
				} else {
					output.WriteString(text[match[0]:match[4]])
					output.WriteString(replacement.String())
					output.WriteString(text[match[5]:match[1]])
				}
			}
			previous = match[1]
		}
		output.WriteString(text[previous:])
		return []byte(output.String()), nil
	}
	expanded, err := expand(importer, source)
	if err != nil {
		return nil, warnings, err
	}
	return expanded, warnings, nil
}

func sassSourceNeedsTrailingSemicolon(source []byte) bool {
	if len(sassVariableDeclaration.FindAllSubmatch(source, -1)) == 0 {
		return false
	}
	lines := strings.Split(string(source), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "*/") {
			continue
		}
		return line == ")"
	}
	return false
}

func sassImportTreeUnresolvedVariable(sources []string, availableVariables map[string]bool) string {
	available := make(map[string]bool, len(availableVariables))
	for name := range availableVariables {
		available[name] = true
	}
	definitions := make(map[string]bool)
	dataBySource := make(map[string][]byte, len(sources))
	for _, source := range uniquePaths(sources) {
		data, err := os.ReadFile(source)
		if err != nil {
			return "<unreadable>"
		}
		dataBySource[source] = data
		for name := range sassVariableNames(data) {
			definitions[name] = true
		}
	}
	for _, source := range uniquePaths(sources) {
		data := dataBySource[source]
		depth := 0
		for _, line := range strings.Split(string(data), "\n") {
			if depth == 0 && sassVariableDeclaration.MatchString(line) {
				for _, match := range sassVariableReference.FindAllSubmatch([]byte(line), -1) {
					name := string(match[1])
					if !definitions[name] && !available[name] {
						return name
					}
				}
			}
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth < 0 {
				depth = 0
			}
		}
	}
	return ""
}

func sassSourceHasNestedMixin(source []byte) bool {
	depth := 0
	for _, line := range strings.Split(string(source), "\n") {
		if depth > 0 && sassMixinDefinition.MatchString(line) {
			return true
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth < 0 {
			depth = 0
		}
	}
	return false
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
			if isPathWithin(importer, appRoot) && !isPathWithin(absolute, appRoot) {
				continue
			}
			info, err := os.Stat(absolute)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				if parent, parentErr := os.Stat(filepath.Dir(absolute)); parentErr == nil && !parent.IsDir() {
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
	return compileClassSassWithInlineImports(appRoot, sources, minify, nil, nil, nil)
}

func compileClassSassWithInlineImports(appRoot string, sources []string, minify bool, inlineImports, skipImports map[string]bool, availableVariables map[string]bool) ([]byte, []string, error) {
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
	entry.WriteString("@function dynamic($value, $args...) { @return $value; }\n")
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, warnings, fmt.Errorf("read Sass source %s: %w", source, err)
		}
		data = normalizeFashionBoxFunctionDefaults(data)
		if sassSourceHasNestedMixin(data) {
			warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: nested Sass mixins are unsupported", source))
			continue
		}
		normalized, dynamicChanged, reason := normalizeFashionDynamic(data)
		if reason != "" {
			if !inlineImports[source] {
				warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: %s", source, reason))
				continue
			}
			normalized = data
			dynamicChanged = false
		}
		if reason := unsupportedFashionSource(normalized); reason != "" && !inlineImports[source] {
			warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: %s", source, reason))
			continue
		}
		if dynamicChanged || inlineImports[source] {
			var importWarnings []string
			normalized, importWarnings, err = inlineSassImports(appRoot, source, normalized, availableVariables, skipImports)
			if err != nil {
				return nil, warnings, fmt.Errorf("normalize Sass imports in %s: %w", source, err)
			}
			warnings = append(warnings, importWarnings...)
			if inlineImports[source] && sassSourceNeedsTrailingSemicolon(normalized) {
				normalized = append(normalized, []byte("\n;\n")...)
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
		match := fashionDynamicAssignment.FindStringIndex(content)
		if match == nil {
			return nil, false, "dynamic() is supported only in simple Sass variable assignments"
		}
		prefixEnd := strings.LastIndex(content[:match[1]], "dynamic(")
		if prefixEnd < 0 {
			return nil, false, "malformed dynamic() Sass variable assignment"
		}
		expressionStart := match[1]
		depth := 1
		quote := byte(0)
		escaped := false
		closeIndex := -1
		for index := expressionStart; index < len(content); index++ {
			current := content[index]
			if quote != 0 {
				if escaped {
					escaped = false
				} else if current == '\\' {
					escaped = true
				} else if current == quote {
					quote = 0
				}
				continue
			}
			if current == '\'' || current == '"' {
				quote = current
				continue
			}
			switch current {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					closeIndex = index
					index = len(content)
				}
			}
		}
		if closeIndex < 0 {
			return nil, false, "unterminated dynamic() Sass variable assignment"
		}
		expression := strings.TrimSpace(content[expressionStart:closeIndex])
		suffix := content[closeIndex+1:]
		if expression == "" || strings.Contains(expression, "dynamic(") || !fashionDynamicSuffix.MatchString(suffix) {
			return nil, false, "dynamic() expression is outside the supported Sass variable assignment subset"
		}
		if !strings.Contains(suffix, ";") {
			if comment := strings.Index(suffix, "//"); comment >= 0 {
				suffix = strings.TrimRight(suffix[:comment], "\t ") + "; " + suffix[comment:]
			} else {
				suffix = strings.TrimRight(suffix, "\t ") + ";"
			}
		}
		output.WriteString(content[:prefixEnd])
		output.WriteString(expression)
		output.WriteString(suffix)
		output.WriteString(lineEnding)
		changed = true
	}
	return []byte(output.String()), changed, ""
}

func unsupportedFashionSource(source []byte) string {
	if _, _, reason := normalizeFashionDynamic(source); reason != "" {
		return reason
	}
	if bytes.Contains(source, []byte("theme-background-image(")) {
		return "theme-background-image requires Sencha Fashion resource context"
	}
	return ""
}
