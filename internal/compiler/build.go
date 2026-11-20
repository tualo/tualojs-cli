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
	FrameworkIncluded    bool
	Toolkit              string
	Theme                string
	Packages             []string
	WatchPaths           []string
	ResourcesPath        string
	ResourceFiles        int
	ResourcePaths        []string
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
	path          string
	name          string
	data          []byte
	originalData  []byte
	priority      int
	framework     bool
	raw           bool
	alwaysInclude bool
	class         []classDeclaration
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
	return BuildWithTargetOptions(appDir, outputDir, selectedToolkit, "", normalizeLetConst, mode, includeSourceMap, minify)
}

func BuildWithTargetOptions(appDir, outputDir, selectedToolkit, selectedPlatform string, normalizeLetConst bool, mode string, includeSourceMap, minify bool) (BuildResult, error) {
	return BuildWithJavaScriptOptions(appDir, outputDir, selectedToolkit, selectedPlatform, normalizeLetConst, false, mode, includeSourceMap, minify)
}

func BuildWithJavaScriptOptions(appDir, outputDir, selectedToolkit, selectedPlatform string, normalizeLetConst, upgradeES5 bool, mode string, includeSourceMap, minify bool) (BuildResult, error) {
	return BuildWithBundleOptions(appDir, outputDir, selectedToolkit, selectedPlatform, normalizeLetConst, upgradeES5, false, mode, includeSourceMap, minify)
}

func BuildWithBundleOptions(appDir, outputDir, selectedToolkit, selectedPlatform string, normalizeLetConst, upgradeES5, includeExtJS bool, mode string, includeSourceMap, minify bool) (BuildResult, error) {
	if mode == "" {
		mode = BuildModeDevelopment
	}
	if mode != BuildModeDevelopment && mode != BuildModeProduction {
		return BuildResult{}, fmt.Errorf("unsupported build mode %q (expected development or production)", mode)
	}
	if selectedPlatform != "" && selectedPlatform != "desktop" && selectedPlatform != "phone" && selectedPlatform != "tablet" {
		return BuildResult{}, fmt.Errorf("unsupported platform %q (expected desktop, phone, or tablet)", selectedPlatform)
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
	config.Classpath, err = expandAppPaths(config.Classpath, root, config.Name, toolkit)
	if err != nil {
		return BuildResult{}, err
	}
	config.Overrides, err = expandAppPaths(config.Overrides, root, config.Name, toolkit)
	if err != nil {
		return BuildResult{}, err
	}
	appSassEtc, err := resolveConfiguredSassSources(root, config.Name, toolkit, sassConfig{Etc: config.Sass.Etc}, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	appSassVar, err := resolveConfiguredSassSources(root, config.Name, toolkit, sassConfig{Var: config.Sass.Var}, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	appSassSrc, err := resolveConfiguredSassSources(root, config.Name, toolkit, sassConfig{Src: config.Sass.Src}, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	sassRoots := append(append([]string{}, config.Classpath...), config.Overrides...)
	classSassSources, err := resolveClassSassSources(root, sassRoots, ignorePatterns)
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
	var optionalPackages []string
	if toolkit != "" {
		optionalPackages = append(optionalPackages, theme)
	}
	appPackages, err := resolveAppPackagesWithOptional(root, requiredPackages, optionalPackages, toolkit, config.Locale)
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
	packageSassEtcRoots := append(append([]packageSassEtcRoot{}, appPackages.sassEtcRoots...), frameworkPackages.sassEtcRoots...)
	packageSassEtcSources, packageThemeSassEtcSources, packageSassEtcWatchPaths, packageSassEtcWarnings, err := resolvePackageSassEtcSources(root, packageSassEtcRoots, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	baseEtcSources := append(append(append([]string{}, appSassEtc...), packageSassEtcSources...), packageThemeSassEtcSources...)
	baseEtcImportPaths, err := resolveSassImportWatchPaths(root, baseEtcSources)
	if err != nil {
		return BuildResult{}, err
	}
	themeSassHelperRoots := append(append([]packageSassEtcRoot{}, appPackages.sassEtcRoots...), frameworkPackages.sassEtcRoots...)
	themeSassHelperRoots = append(themeSassHelperRoots, packageSassEtcHelperRoots(appPackages.sdkRoot, baseEtcImportPaths)...)
	themeSassHelperSources, err := resolveThemeSassHelperSources(root, packageThemeSassEtcSources, themeSassHelperRoots, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	etcSassSources := append(append(append(append([]string{}, appSassEtc...), packageSassEtcSources...), themeSassHelperSources...), packageThemeSassEtcSources...)
	etcSassImportWatchPaths, err := resolveSassImportWatchPaths(root, etcSassSources)
	if err != nil {
		return BuildResult{}, err
	}
	etcSassVariableSources := append(append([]string{}, etcSassSources...), etcSassImportWatchPaths...)
	appEtcVariables, err := collectSassVariableNames(etcSassVariableSources)
	if err != nil {
		return BuildResult{}, err
	}
	packageSassVarRoots := uniquePaths(append(append([]string{}, appPackages.sassVarRoots...), frameworkPackages.sassVarRoots...))
	packageSassVarSources, packageSassVarWatchPaths, packageSassVariables, packageSassVarWarnings, err := resolvePackageSassVariables(root, packageSassVarRoots, appEtcVariables, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	appVarVariables, err := collectSassVariableNames(appSassVar)
	if err != nil {
		return BuildResult{}, err
	}
	for name := range appVarVariables {
		packageSassVariables[name] = true
	}
	packageSassRoots := uniquePackageSassSourceRoots(append(append([]packageSassSourceRoot{}, appPackages.sassRoots...), frameworkPackages.sassRoots...))
	packageSassSources, packageSassWatchPaths, packageSassWarnings, err := resolvePackageClassSassSources(root, packageSassRoots, packageSassVariables, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	sassSources := uniquePaths(append(append(append(append(append(append(append(append(appSassEtc, themeSassHelperSources...), packageSassEtcSources...), packageSassVarSources...), packageThemeSassEtcSources...), appSassVar...), packageSassSources...), appSassSrc...), classSassSources...))
	if len(sassSources) > 0 && toolkit == "" {
		return BuildResult{}, fmt.Errorf("sass compilation requires a selected toolkit; pass -toolkit classic or -toolkit modern")
	}
	sassImportWatchPaths, err := resolveSassImportWatchPaths(root, sassSources)
	if err != nil {
		return BuildResult{}, err
	}
	sassMixinSources := append(append([]string{}, sassSources...), sassImportWatchPaths...)
	sassMixins, err := collectSassMixinNames(sassMixinSources)
	if err != nil {
		return BuildResult{}, err
	}
	sassMixins["border-radius"] = true
	sassMixins["border-top-left-radius"] = true
	sassMixins["border-top-right-radius"] = true
	sassMixins["border-bottom-right-radius"] = true
	sassMixins["border-bottom-left-radius"] = true
	sassMixins["opacity"] = true
	sassMixins["user-select"] = true
	sassMixins["box-shadow"] = true
	sassMixins["rotate"] = true
	sassMixins["scale"] = true
	sassSources, sassMixinWarnings, err := filterSassSourcesByMixins(sassSources, sassMixins)
	if err != nil {
		return BuildResult{}, err
	}
	sassIncludeDefaults, err := resolveSassIncludeDefaults(append(append([]string{}, sassSources...), sassImportWatchPaths...), packageSassVariables)
	if err != nil {
		return BuildResult{}, err
	}
	runtimePath, err := frameworkRuntimePath(config, appPackages.sdkRoot, toolkit)
	if err != nil {
		return BuildResult{}, err
	}
	var embeddedFramework *sourceFile
	if includeExtJS {
		if runtimePath == "" {
			return BuildResult{}, fmt.Errorf("including Ext JS in app.js requires an Ext JS SDK and a selected toolkit")
		}
		runtimeData, err := os.ReadFile(runtimePath)
		if err != nil {
			return BuildResult{}, fmt.Errorf("read Ext JS runtime %s: %w", runtimePath, err)
		}
		runtimeName := filepath.ToSlash(filepath.Join("framework", filepath.Base(runtimePath)))
		embeddedFramework = &sourceFile{
			path:         runtimePath,
			name:         runtimeName,
			data:         runtimeData,
			originalData: append([]byte(nil), runtimeData...),
			framework:    true,
			raw:          true,
		}
	}
	entryPath, err := bundleEntry(config)
	if err != nil {
		return BuildResult{}, err
	}
	entryPaths, err := expandAppPaths([]string{entryPath}, root, config.Name, toolkit)
	if err != nil {
		return BuildResult{}, err
	}
	entryPath = entryPaths[0]
	entryPath, err = projectPath(root, entryPath)
	if err != nil {
		return BuildResult{}, fmt.Errorf("JavaScript entry: %w", err)
	}
	scripts, scriptWatchPaths, err := resolveScripts(root, config.Name, config.JavaScript, toolkit, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	entryOriginalData, err := os.ReadFile(entryPath)
	if err != nil {
		return BuildResult{}, fmt.Errorf("read JavaScript entry %s: %w", entryPath, err)
	}
	entrySource, err := preprocessSenchaForTarget(string(entryOriginalData), mode, toolkit, selectedPlatform)
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
	for _, script := range scripts {
		if !script.includeInBundle {
			continue
		}
		relative, err := filepath.Rel(root, script.source)
		if err != nil {
			return BuildResult{}, err
		}
		paths = append(paths, classSourcePath{path: relative, base: root, alwaysInclude: true})
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
	watchPaths = append(watchPaths, sassSources...)
	watchPaths = append(watchPaths, packageSassEtcWatchPaths...)
	watchPaths = append(watchPaths, packageSassVarWatchPaths...)
	watchPaths = append(watchPaths, packageSassWatchPaths...)
	watchPaths = append(watchPaths, sassImportWatchPaths...)
	if embeddedFramework != nil {
		watchPaths = append(watchPaths, embeddedFramework.path)
	}
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
	sources, classes, overrides, aliases, frameworkFiles, err := loadClasses(paths, mode, root, toolkit, selectedPlatform, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	includeSources := make([]sourceFile, 0)
	for _, source := range sources {
		if !source.alwaysInclude {
			continue
		}
		includeSources = append(includeSources, source)
	}
	sort.SliceStable(includeSources, func(i, j int) bool {
		if includeSources[i].priority == includeSources[j].priority {
			return includeSources[i].name < includeSources[j].name
		}
		return includeSources[i].priority < includeSources[j].priority
	})
	for _, source := range includeSources {
		for _, declaration := range source.class {
			if declaration.override == "" {
				appDependencies = append(appDependencies, declaration.name)
			}
		}
	}
	ordered, warnings, frameworkClasses, err := orderSources(config.Namespace, appDependencies, classes, overrides, aliases)
	if err != nil {
		return BuildResult{}, err
	}
	warnings = append(warnings, appPackages.sassWarnings...)
	warnings = append(warnings, frameworkPackages.sassWarnings...)
	warnings = append(warnings, packageSassEtcWarnings...)
	warnings = append(warnings, packageSassVarWarnings...)
	warnings = append(warnings, packageSassWarnings...)
	warnings = append(warnings, sassMixinWarnings...)
	orderedPaths := make(map[string]bool, len(ordered))
	for _, source := range ordered {
		orderedPaths[source.path] = true
	}
	var includeOnly []sourceFile
	for _, source := range sources {
		if source.alwaysInclude && !orderedPaths[source.path] {
			includeOnly = append(includeOnly, source)
		}
	}
	sort.SliceStable(includeOnly, func(i, j int) bool {
		if includeOnly[i].priority == includeOnly[j].priority {
			return includeOnly[i].name < includeOnly[j].name
		}
		return includeOnly[i].priority < includeOnly[j].priority
	})
	ordered = append(ordered, includeOnly...)
	if embeddedFramework != nil {
		ordered = append([]sourceFile{*embeddedFramework}, ordered...)
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
	bundle, sourceMapData, err := assembleBundle(ordered, normalizeLetConst, upgradeES5, includeSourceMap, minify)
	if err != nil {
		return BuildResult{}, err
	}
	inlineSassImports := make(map[string]bool, len(packageThemeSassEtcSources)+len(themeSassHelperSources))
	for _, source := range packageSassEtcSources {
		inlineSassImports[source] = true
	}
	for _, source := range packageThemeSassEtcSources {
		inlineSassImports[source] = true
	}
	skipThemeImports := make(map[string]bool, len(themeSassHelperSources))
	for _, source := range themeSassHelperSources {
		inlineSassImports[source] = true
		skipThemeImports[source] = true
	}
	classSassCSS, classSassWarnings, err := compileClassSassWithInlineImports(root, sassSources, minify, inlineSassImports, skipThemeImports, packageSassVariables, sassIncludeDefaults)
	if err != nil {
		return BuildResult{}, err
	}
	warnings = append(warnings, classSassWarnings...)
	outputPath := filepath.Join(outputRoot, "app.js")
	if embeddedFramework != nil && filepath.Clean(outputPath) == filepath.Clean(embeddedFramework.path) {
		return BuildResult{}, fmt.Errorf("output bundle would overwrite Ext JS runtime %s", embeddedFramework.path)
	}
	if outputPath == entryPath {
		return BuildResult{}, fmt.Errorf("output bundle would overwrite JavaScript entry %s", entryPath)
	}
	for _, source := range sources {
		if outputPath == source.path {
			return BuildResult{}, fmt.Errorf("output bundle would overwrite class source %s", source.path)
		}
	}
	resourcePaths, err := copyResources(resources, outputRoot, root, ignorePatterns)
	if err != nil {
		return BuildResult{}, err
	}
	resourceFiles := len(resourcePaths)
	copiedStylesheets, err := copyStylesheets(styleSheets, outputRoot, outputPath)
	if err != nil {
		return BuildResult{}, err
	}
	if len(classSassCSS) > 0 {
		const generatedSassURL = "app-sass.css"
		generatedSassOutput := filepath.ToSlash(filepath.Join("resources", toolkit, "resources", generatedSassURL))
		for _, stylesheet := range copiedStylesheets {
			stylesheetURL := filepath.ToSlash(stylesheet.URL)
			if stylesheetURL == generatedSassURL || (!stylesheet.Remote && stylesheetURL == generatedSassOutput) {
				return BuildResult{}, fmt.Errorf("configured stylesheet output %s conflicts with generated class Sass", generatedSassURL)
			}
		}
		generatedSassPath := filepath.Join(outputRoot, "resources", toolkit, "resources", generatedSassURL)
		if err := os.MkdirAll(filepath.Dir(generatedSassPath), 0o755); err != nil {
			return BuildResult{}, fmt.Errorf("create generated Sass output directory: %w", err)
		}
		if err := writeFileAtomic(generatedSassPath, classSassCSS); err != nil {
			return BuildResult{}, fmt.Errorf("write generated Sass stylesheet %s: %w", generatedSassPath, err)
		}
		copiedStylesheets = append(copiedStylesheets, StyleSheet{URL: generatedSassURL, Path: generatedSassPath})
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
		FrameworkIncluded:    includeExtJS,
		Toolkit:              toolkit,
		Theme:                theme,
		Packages:             appPackages.packages,
		WatchPaths:           watchPaths,
		ResourcesPath:        filepath.Join(outputRoot, "resources"),
		ResourceFiles:        resourceFiles,
		ResourcePaths:        resourcePaths,
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
	temporary, err := os.CreateTemp(filepath.Dir(path), ".tualojs-*.tmp")
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

func loadClasses(paths []classSourcePath, mode, appRoot, toolkit, platform string, ignore []*regexp.Regexp) ([]sourceFile, map[string]sourceFile, map[string][]overrideDeclaration, map[string][]string, map[string]string, error) {
	var sources []sourceFile
	seenSources := make(map[string]int)
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
				if err := appendUniqueSource(sourcePath.base, path, sourcePath.prefix, priority, sourcePath.framework, sourcePath.alwaysInclude, mode, toolkit, platform, &sources, seenSources); err != nil {
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
			return appendUniqueSource(sourcePath.base, filePath, sourcePath.prefix, priority, sourcePath.framework, sourcePath.alwaysInclude, mode, toolkit, platform, &sources, seenSources)
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
				overrides[target] = append(overrides[target], overrideDeclaration(candidate))
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

func appendUniqueSource(root, path, prefix string, priority int, framework, alwaysInclude bool, mode, toolkit, platform string, sources *[]sourceFile, seen map[string]int) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if index, exists := seen[absolute]; exists {
		source := &(*sources)[index]
		source.framework = source.framework || framework
		source.alwaysInclude = source.alwaysInclude || alwaysInclude
		if priority < source.priority {
			source.priority = priority
		}
		return nil
	}
	if err := appendSource(root, absolute, prefix, priority, framework, alwaysInclude, mode, toolkit, platform, sources); err != nil {
		return err
	}
	seen[absolute] = len(*sources) - 1
	return nil
}

func appendSource(root, path, prefix string, priority int, framework, alwaysInclude bool, mode, toolkit, platform string, sources *[]sourceFile) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read source %s: %w", path, err)
	}
	originalData := append([]byte(nil), data...)
	processed, err := preprocessSenchaForTarget(string(data), mode, toolkit, platform)
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
	*sources = append(*sources, sourceFile{path: path, name: prefix + filepath.ToSlash(relative), data: data, originalData: originalData, priority: priority, framework: framework, alwaysInclude: alwaysInclude, class: declarations})
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
