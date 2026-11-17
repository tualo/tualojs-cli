package compiler

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	BuildModeDevelopment = "development"
	BuildModeProduction  = "production"
)

type BuildResult struct {
	AppName              string
	AppDir               string
	SourceCount          int
	OutputPath           string
	Warnings             []string
	FrameworkVersion     string
	FrameworkSDKPath     string
	FrameworkRuntimePath string
	Toolkit              string
	Theme                string
	Packages             []string
	WatchPaths           []string
	ResourcesPath        string
	ResourceFiles        int
	StyleSheets          []StyleSheet
	Scripts              []ScriptAsset
	FrameworkClasses     []string
	FrameworkFiles       map[string]string
	SourceFiles          map[string]string
	Mode                 string
	SourceMapPath        string
}

type ScriptAsset struct {
	URL    string
	Remote bool
}

type sourceFile struct {
	path         string
	name         string
	data         []byte
	originalData []byte
	priority     int
	framework    bool
	class        []classDeclaration
}

type overrideDeclaration struct {
	source      sourceFile
	declaration classDeclaration
}

type classCandidate struct {
	source      sourceFile
	declaration classDeclaration
}

func Build(appDir, outputDir string) (BuildResult, error) {
	return BuildWithMode(appDir, outputDir, "", false, BuildModeDevelopment)
}

func BuildWithToolkit(appDir, outputDir, selectedToolkit string) (BuildResult, error) {
	return BuildWithMode(appDir, outputDir, selectedToolkit, false, BuildModeDevelopment)
}

func BuildWithOptions(appDir, outputDir, selectedToolkit string, normalizeLetConst bool) (BuildResult, error) {
	return BuildWithMode(appDir, outputDir, selectedToolkit, normalizeLetConst, BuildModeDevelopment)
}

func BuildWithMode(appDir, outputDir, selectedToolkit string, normalizeLetConst bool, mode string) (BuildResult, error) {
	return BuildWithSourceMap(appDir, outputDir, selectedToolkit, normalizeLetConst, mode, false)
}

func BuildWithSourceMap(appDir, outputDir, selectedToolkit string, normalizeLetConst bool, mode string, includeSourceMap bool) (BuildResult, error) {
	return BuildWithOutputOptions(appDir, outputDir, selectedToolkit, normalizeLetConst, mode, includeSourceMap, false)
}

func BuildWithOutputOptions(appDir, outputDir, selectedToolkit string, normalizeLetConst bool, mode string, includeSourceMap, minify bool) (BuildResult, error) {
	if mode == "" {
		mode = BuildModeDevelopment
	}
	if mode != BuildModeDevelopment && mode != BuildModeProduction {
		return BuildResult{}, fmt.Errorf("unsupported build mode %q (expected development or production)", mode)
	}
	config, root, err := loadAppConfig(appDir)
	if err != nil {
		return BuildResult{}, err
	}
	ignorePatterns, err := compileIgnorePatterns(config.Ignore)
	if err != nil {
		return BuildResult{}, err
	}
	toolkit := selectedToolkit
	if toolkit == "" {
		toolkit = config.Toolkit
	}
	if toolkit != "" && toolkit != "classic" && toolkit != "modern" {
		return BuildResult{}, fmt.Errorf("unsupported toolkit %q (expected classic or modern)", toolkit)
	}
	requiredPackages := append([]string{}, config.Requires...)
	if toolkit == "classic" {
		requiredPackages = append(requiredPackages, config.Classic.Requires...)
	} else if toolkit == "modern" {
		requiredPackages = append(requiredPackages, config.Modern.Requires...)
	}
	if profile, exists := config.Builds[toolkit]; exists {
		requiredPackages = append(requiredPackages, profile.Requires...)
	}
	theme := config.Theme
	if profile, exists := config.Builds[toolkit]; exists && profile.Toolkit == toolkit && profile.Theme != "" {
		theme = profile.Theme
	}
	if theme == "" {
		if toolkit == "classic" {
			theme = "theme-triton"
		} else {
			theme = "theme-material"
		}
	}
	config.Classpath, err = expandToolkitPaths(config.Classpath, toolkit)
	if err != nil {
		return BuildResult{}, err
	}
	config.Overrides, err = expandToolkitPaths(config.Overrides, toolkit)
	if err != nil {
		return BuildResult{}, err
	}
	if toolkit == "" && len(config.Requires) > 0 {
		return BuildResult{}, fmt.Errorf("app package resolution requires a toolkit; pass -toolkit classic or -toolkit modern")
	}
	resources, resourceWatchPaths, err := resolveResources(root, config.Name, mode, config.Resources, toolkit, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	styleSheets, styleWatchPaths, err := resolveStylesheets(root, config, toolkit, mode, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	appPackages, err := resolveAppPackages(root, requiredPackages, toolkit, config.Locale)
	if err != nil {
		return BuildResult{}, err
	}
	resources = append(resources, appPackages.resources...)
	var frameworkPackages packageResolution
	if appPackages.sdkRoot != "" {
		if toolkit == "" {
			return BuildResult{}, fmt.Errorf("framework class resolution requires a toolkit; pass -toolkit classic or -toolkit modern")
		}
		frameworkPackages, err = resolveAppPackages(root, []string{toolkit}, toolkit, config.Locale)
		if err != nil {
			return BuildResult{}, fmt.Errorf("resolve framework toolkit classes: %w", err)
		}
	}
	runtimePath, err := frameworkRuntimePath(config, appPackages.sdkRoot, toolkit)
	if err != nil {
		return BuildResult{}, err
	}
	entryPath, err := bundleEntry(config)
	if err != nil {
		return BuildResult{}, err
	}
	entryPath, err = projectPath(root, entryPath)
	if err != nil {
		return BuildResult{}, fmt.Errorf("JavaScript entry: %w", err)
	}
	scripts, scriptWatchPaths, err := resolveScripts(root, config.JavaScript, toolkit, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	entryOriginalData, err := os.ReadFile(entryPath)
	if err != nil {
		return BuildResult{}, fmt.Errorf("read JavaScript entry %s: %w", entryPath, err)
	}
	entrySource, err := preprocessSencha(string(entryOriginalData), mode)
	if err != nil {
		return BuildResult{}, fmt.Errorf("preprocess JavaScript entry %s: %w", entryPath, err)
	}
	entryData := []byte(entrySource)
	_, appDependencies, sourceDependencies, err := parseExtMetadata(string(entryData))
	if err != nil {
		return BuildResult{}, fmt.Errorf("parse JavaScript entry %s: %w", entryPath, err)
	}
	appDependencies = append(appDependencies, sourceDependencies...)
	entryLoaderPaths, err := parseLoaderPaths(string(entryData))
	if err != nil {
		return BuildResult{}, fmt.Errorf("parse JavaScript entry loader paths: %w", err)
	}

	paths := make([]classSourcePath, 0, len(config.Classpath)+len(config.Overrides)+len(appPackages.classpaths)+len(appPackages.overrides)+len(frameworkPackages.classpaths)+len(frameworkPackages.overrides))
	for _, path := range config.Classpath {
		paths = append(paths, classSourcePath{path: path, base: root})
	}
	for _, path := range config.Overrides {
		paths = append(paths, classSourcePath{path: path, base: root, optional: true})
	}
	paths = append(paths, appPackages.classpaths...)
	paths = append(paths, appPackages.overrides...)
	for _, sourcePath := range append(frameworkPackages.classpaths, frameworkPackages.overrides...) {
		sourcePath.framework = true
		paths = append(paths, sourcePath)
	}
	loaderPaths, err := discoverLoaderPaths(paths)
	if err != nil {
		return BuildResult{}, err
	}
	loaderPaths = uniqueLoaderMappings(append(entryLoaderPaths, loaderPaths...))
	paths = append(paths, resolveLoaderPathSources(root, config.Namespace, loaderPaths, paths)...)
	watchPaths := append([]string{}, appPackages.watchPaths...)
	watchPaths = append(watchPaths, frameworkPackages.watchPaths...)
	watchPaths = append(watchPaths, resourceWatchPaths...)
	watchPaths = append(watchPaths, styleWatchPaths...)
	watchPaths = append(watchPaths, scriptWatchPaths...)
	watchPaths = append(watchPaths, filepath.Join(root, "app.json"), filepath.Join(root, "workspace.json"), entryPath)
	for _, sourcePath := range paths {
		if sourcePath.framework {
			continue
		}
		resolved, pathErr := projectPath(sourcePath.base, sourcePath.path)
		if pathErr == nil {
			watchPaths = append(watchPaths, resolved)
		}
	}
	watchPaths = uniquePaths(watchPaths)
	sources, classes, overrides, aliases, frameworkFiles, err := loadClasses(paths, mode, root, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	ordered, warnings, frameworkClasses, err := orderSources(config.Namespace, appDependencies, classes, overrides, aliases)
	if err != nil {
		return BuildResult{}, err
	}
	entryName, _ := filepath.Rel(root, entryPath)
	entry := sourceFile{path: entryPath, name: filepath.ToSlash(entryName), data: entryData, originalData: entryOriginalData}
	ordered = append(ordered, entry)
	sourceFiles := make(map[string]string, len(ordered))
	for _, source := range ordered {
		sourceFiles[source.name] = source.path
	}

	outputRoot := outputDir
	if !filepath.IsAbs(outputRoot) {
		outputRoot = filepath.Join(root, outputRoot)
	}
	outputRoot, err = filepath.Abs(outputRoot)
	if err != nil {
		return BuildResult{}, fmt.Errorf("resolve output directory: %w", err)
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return BuildResult{}, fmt.Errorf("create output directory: %w", err)
	}
	bundle, sourceMapData, err := assembleBundle(ordered, normalizeLetConst, includeSourceMap, minify)
	if err != nil {
		return BuildResult{}, err
	}
	outputPath := filepath.Join(outputRoot, "app.js")
	if outputPath == entryPath {
		return BuildResult{}, fmt.Errorf("output bundle would overwrite JavaScript entry %s", entryPath)
	}
	for _, source := range sources {
		if outputPath == source.path {
			return BuildResult{}, fmt.Errorf("output bundle would overwrite class source %s", source.path)
		}
	}
	resourceFiles, err := copyResources(resources, outputRoot, root, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	copiedStylesheets, err := copyStylesheets(styleSheets, outputRoot, outputPath)
	if err != nil {
		return BuildResult{}, err
	}
	jsAssets, err := copyScripts(scripts, outputRoot, outputPath)
	if err != nil {
		return BuildResult{}, err
	}
	if err := writeFileAtomic(outputPath, bundle); err != nil {
		return BuildResult{}, fmt.Errorf("write bundle %s: %w", outputPath, err)
	}
	sourceMapPath := ""
	if includeSourceMap {
		sourceMapPath = outputPath + ".map"
		if err := writeFileAtomic(sourceMapPath, sourceMapData); err != nil {
			return BuildResult{}, fmt.Errorf("write source map %s: %w", sourceMapPath, err)
		}
	}
	return BuildResult{
		AppName:              config.Name,
		AppDir:               root,
		SourceCount:          len(ordered),
		OutputPath:           outputPath,
		Warnings:             warnings,
		FrameworkVersion:     appPackages.version,
		FrameworkSDKPath:     appPackages.sdkRoot,
		FrameworkRuntimePath: runtimePath,
		Toolkit:              toolkit,
		Theme:                theme,
		Packages:             appPackages.packages,
		WatchPaths:           watchPaths,
		ResourcesPath:        filepath.Join(outputRoot, "resources"),
		ResourceFiles:        resourceFiles,
		StyleSheets:          copiedStylesheets,
		Scripts:              jsAssets,
		FrameworkClasses:     frameworkClasses,
		FrameworkFiles:       frameworkFiles,
		SourceFiles:          sourceFiles,
		Mode:                 mode,
		SourceMapPath:        sourceMapPath,
	}, nil
}

func discoverLoaderPaths(paths []classSourcePath) ([]loaderPathMapping, error) {
	var mappings []loaderPathMapping
	for _, sourcePath := range paths {
		if sourcePath.framework {
			continue
		}
		path, err := projectPath(sourcePath.base, sourcePath.path)
		if err != nil {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			if sourcePath.optional && os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("inspect loader-path source %s: %w", path, err)
		}
		parseFile := func(file string) error {
			if !strings.EqualFold(filepath.Ext(file), ".js") {
				return nil
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			found, err := parseLoaderPaths(string(data))
			if err != nil {
				return fmt.Errorf("parse loader paths in %s: %w", file, err)
			}
			mappings = append(mappings, found...)
			return nil
		}
		if !info.IsDir() {
			if err := parseFile(path); err != nil {
				return nil, err
			}
			continue
		}
		if err := filepath.WalkDir(path, func(file string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			return parseFile(file)
		}); err != nil {
			return nil, fmt.Errorf("scan loader paths under %s: %w", path, err)
		}
	}
	return uniqueLoaderMappings(mappings), nil
}

func resolveLoaderPathSources(root, namespace string, mappings []loaderPathMapping, existing []classSourcePath) []classSourcePath {
	known := make(map[string]bool)
	for _, path := range existing {
		resolved, err := projectPath(path.base, path.path)
		if err == nil {
			known[resolved] = true
		}
	}
	var paths []classSourcePath
	for _, mapping := range mappings {
		if mapping.Namespace != namespace && !strings.HasPrefix(mapping.Namespace, namespace+".") {
			continue
		}
		if mapping.Path == "" || strings.Contains(mapping.Path, "://") {
			continue
		}
		resolved, err := projectPath(root, mapping.Path)
		if err != nil || resolved == root || known[resolved] {
			continue
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			continue
		}
		known[resolved] = true
		relative, err := filepath.Rel(root, resolved)
		if err == nil {
			paths = append(paths, classSourcePath{path: relative, base: root, optional: true})
		}
	}
	return paths
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if !seen[path] {
			seen[path] = true
			unique = append(unique, path)
		}
	}
	return unique
}

func writeFileAtomic(path string, data []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".extjsc-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temporaryName, 0o644); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

func bundleEntry(config appConfig) (string, error) {
	var entries []string
	for _, entry := range config.JavaScript {
		if entry.Bundle {
			entries = append(entries, entry.Path)
		}
	}
	if len(entries) != 1 {
		return "", fmt.Errorf("app.json must define exactly one JavaScript entry with bundle:true (found %d)", len(entries))
	}
	return entries[0], nil
}

func loadClasses(paths []classSourcePath, mode, appRoot string, ignore []*regexp.Regexp) ([]sourceFile, map[string]sourceFile, map[string][]overrideDeclaration, map[string][]string, map[string]string, error) {
	var sources []sourceFile
	for priority, sourcePath := range paths {
		path, err := projectPath(sourcePath.base, sourcePath.path)
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("source path: %w", err)
		}
		info, err := os.Stat(path)
		if err != nil {
			if sourcePath.optional && os.IsNotExist(err) {
				continue
			}
			return nil, nil, nil, nil, nil, fmt.Errorf("source path %q: %w", sourcePath.path, err)
		}
		if !info.IsDir() {
			if isIgnoredProjectPath(appRoot, path, ignore) {
				continue
			}
			if strings.EqualFold(filepath.Ext(path), ".js") {
				if err := appendSource(sourcePath.base, path, sourcePath.prefix, priority, sourcePath.framework, mode, &sources); err != nil {
					return nil, nil, nil, nil, nil, err
				}
			}
			continue
		}
		err = filepath.WalkDir(path, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".js") {
				if entry.IsDir() && isIgnoredProjectPath(appRoot, filePath, ignore) {
					return filepath.SkipDir
				}
				return nil
			}
			if isIgnoredProjectPath(appRoot, filePath, ignore) {
				return nil
			}
			return appendSource(sourcePath.base, filePath, sourcePath.prefix, priority, sourcePath.framework, mode, &sources)
		})
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("scan source path %q: %w", sourcePath.path, err)
		}
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].name < sources[j].name })

	candidatesByName := make(map[string][]classCandidate)
	for _, source := range sources {
		for _, declaration := range source.class {
			candidatesByName[declaration.name] = append(candidatesByName[declaration.name], classCandidate{source: source, declaration: declaration})
		}
	}

	classes := make(map[string]sourceFile)
	overrides := make(map[string][]overrideDeclaration)
	aliases := make(map[string][]string)
	frameworkFiles := make(map[string]string)
	names := make([]string, 0, len(candidatesByName))
	for name := range candidatesByName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		candidates := candidatesByName[name]
		selectedPath, err := selectClassCandidate(name, candidates)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		for _, candidate := range candidates {
			if candidate.source.path != selectedPath {
				continue
			}
			if candidate.declaration.override != "" {
				target := candidate.declaration.override
				overrides[target] = append(overrides[target], overrideDeclaration{source: candidate.source, declaration: candidate.declaration})
				if candidate.source.framework {
					frameworkFiles[candidate.declaration.name] = candidate.source.path
				}
			} else {
				_, exists := classes[name]
				if !exists || candidate.source.framework {
					classes[name] = candidate.source
				}
				if candidate.source.framework {
					frameworkFiles[name] = candidate.source.path
				}
				for _, alias := range candidate.declaration.aliases {
					aliases[alias] = append(aliases[alias], name)
				}
			}
		}
	}
	for alias := range aliases {
		aliases[alias] = uniqueStrings(aliases[alias])
		sort.Strings(aliases[alias])
	}
	return sources, classes, overrides, aliases, frameworkFiles, nil
}

func selectClassCandidate(name string, candidates []classCandidate) (string, error) {
	paths := make(map[string]bool)
	for _, candidate := range candidates {
		paths[candidate.source.path] = true
	}
	if len(paths) == 1 {
		for path := range paths {
			return path, nil
		}
	}
	className := name
	if index := strings.LastIndex(className, "."); index >= 0 {
		className = className[index+1:]
	}
	canonicalPaths := make(map[string]bool)
	for _, candidate := range candidates {
		if strings.EqualFold(filepath.Base(candidate.source.path), className+".js") {
			canonicalPaths[candidate.source.path] = true
		}
	}
	if len(canonicalPaths) > 0 {
		candidates = filterCandidates(candidates, canonicalPaths)
	}
	selected, ok := preferredCandidate(candidates)
	if ok {
		return selected, nil
	}
	files := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		files = append(files, candidate.source.name)
	}
	sort.Strings(files)
	return "", fmt.Errorf("duplicate class %q in %s", name, strings.Join(files, " and "))
}

func filterCandidates(candidates []classCandidate, paths map[string]bool) []classCandidate {
	filtered := make([]classCandidate, 0, len(paths))
	for _, candidate := range candidates {
		if paths[candidate.source.path] {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func preferredCandidate(candidates []classCandidate) (string, bool) {
	if len(candidates) == 0 {
		return "", false
	}
	minPriority := candidates[0].source.priority
	for _, candidate := range candidates[1:] {
		if candidate.source.priority < minPriority {
			minPriority = candidate.source.priority
		}
	}
	var preferred []classCandidate
	for _, candidate := range candidates {
		if candidate.source.priority == minPriority {
			preferred = append(preferred, candidate)
		}
	}
	if len(preferred) == 1 {
		return preferred[0].source.path, true
	}
	first := preferred[0].source.data
	for _, candidate := range preferred[1:] {
		if !bytes.Equal(first, candidate.source.data) {
			return "", false
		}
	}
	sort.Slice(preferred, func(i, j int) bool { return preferred[i].source.name < preferred[j].source.name })
	return preferred[0].source.path, true
}

func appendSource(root, path, prefix string, priority int, framework bool, mode string, sources *[]sourceFile) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read source %s: %w", path, err)
	}
	originalData := append([]byte(nil), data...)
	processed, err := preprocessSencha(string(data), mode)
	if err != nil {
		return fmt.Errorf("preprocess source %s: %w", path, err)
	}
	data = []byte(processed)
	declarations, _, sourceDependencies, err := parseExtMetadata(string(data))
	if err != nil {
		return fmt.Errorf("parse source %s: %w", path, err)
	}
	for i := range declarations {
		declarations[i].dependencies = append(declarations[i].dependencies, sourceDependencies...)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	*sources = append(*sources, sourceFile{path: path, name: prefix + filepath.ToSlash(relative), data: data, originalData: originalData, priority: priority, framework: framework, class: declarations})
	return nil
}

func orderSources(namespace string, appDependencies []string, classes map[string]sourceFile, overrides map[string][]overrideDeclaration, aliases map[string][]string) ([]sourceFile, []string, []string, error) {
	state := make(map[string]uint8)
	var ordered []sourceFile
	addedFiles := make(map[string]bool)
	unresolved := make(map[string]bool)
	frameworkLoaded := make(map[string]bool)
	var frameworkClasses []string
	var visit func(string, []string) error
	visit = func(name string, chain []string) error {
		if targets := aliases[name]; len(targets) > 0 {
			for _, target := range targets {
				if state[target] == 1 {
					continue
				}
				if err := visit(target, chain); err != nil {
					return err
				}
			}
			return nil
		}
		if strings.Contains(name, "*") {
			for _, match := range matchingClasses(name, namespace, classes) {
				if state[match] == 1 {
					continue
				}
				if err := visit(match, chain); err != nil {
					return err
				}
			}
			return nil
		}
		source, exists := classes[name]
		if state[name] == 2 {
			return nil
		}
		if state[name] == 1 {
			return fmt.Errorf("class dependency cycle: %s -> %s", strings.Join(chain, " -> "), name)
		}
		state[name] = 1
		var deferredUses []string
		if exists {
			if source.framework {
				if !frameworkLoaded[name] {
					frameworkClasses = append(frameworkClasses, name)
					frameworkLoaded[name] = true
				}
			} else {
				declaration := declarationFor(source, name)
				deferredUses = append(deferredUses, declaration.uses...)
				for _, dependency := range declaration.dependencies {
					if dependency == name {
						continue
					}
					if err := visit(dependency, append(chain, name)); err != nil {
						return err
					}
				}
				if !addedFiles[source.path] {
					ordered = append(ordered, source)
					addedFiles[source.path] = true
				}
			}
		} else if name == namespace || strings.HasPrefix(name, namespace+".") {
			unresolved[name] = true
			state[name] = 2
			return nil
		}
		for _, override := range overrides[name] {
			for _, dependency := range override.declaration.dependencies {
				if err := visit(dependency, append(chain, name)); err != nil {
					return err
				}
			}
			if override.source.framework {
				if !frameworkLoaded[override.declaration.name] {
					frameworkClasses = append(frameworkClasses, override.declaration.name)
					frameworkLoaded[override.declaration.name] = true
				}
			} else if !addedFiles[override.source.path] {
				ordered = append(ordered, override.source)
				addedFiles[override.source.path] = true
			}
			if !override.source.framework {
				deferredUses = append(deferredUses, override.declaration.uses...)
			}
		}
		state[name] = 2
		for _, usedClass := range deferredUses {
			if err := visit(usedClass, append(chain, name)); err != nil {
				return err
			}
		}
		return nil
	}
	for _, dependency := range appDependencies {
		if err := visit(dependency, nil); err != nil {
			return nil, nil, nil, err
		}
	}
	warnings := make([]string, 0, len(unresolved))
	for name := range unresolved {
		warnings = append(warnings, fmt.Sprintf("application class %q was not found in the selected classpaths; it may be loaded dynamically", name))
	}
	sort.Strings(warnings)
	return ordered, warnings, frameworkClasses, nil
}

func matchingClasses(pattern, namespace string, classes map[string]sourceFile) []string {
	if strings.Count(pattern, "*") != 1 || !strings.HasSuffix(pattern, "*") {
		return nil
	}
	prefix := strings.TrimSuffix(pattern, "*")
	if prefix != namespace+"." && !strings.HasPrefix(prefix, namespace+".") && prefix != "Ext." && !strings.HasPrefix(prefix, "Ext.") {
		return nil
	}
	var matches []string
	for name := range classes {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	return matches
}

func declarationFor(source sourceFile, name string) classDeclaration {
	combined := classDeclaration{name: name}
	for _, candidate := range source.class {
		if candidate.name == name && candidate.override == "" {
			combined.dependencies = append(combined.dependencies, candidate.dependencies...)
			combined.uses = append(combined.uses, candidate.uses...)
		}
	}
	return combined
}
