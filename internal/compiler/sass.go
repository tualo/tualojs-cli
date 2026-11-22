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
var sassIncludeVariableReference = regexp.MustCompile(`\$(include-[a-zA-Z0-9_-]+)`)
var sassFunctionDefinition = regexp.MustCompile(`(?m)^[\t ]*@function[\t ]+([a-zA-Z_-][a-zA-Z0-9_-]*)`)
var sassScopedCallableDefinition = regexp.MustCompile(`(?m)^[\t ]*@(function|mixin)[\t ]+[a-zA-Z_-][a-zA-Z0-9_-]*[\t ]*\(`)
var fashionMinMaxFunctionDefinition = regexp.MustCompile(`(?m)^[\t ]*@function[\t ]+(min|max)[\t ]*\(`)
var fontAwesomeEscapedIconInterpolation = regexp.MustCompile(`\\#\{\s*\$icon\s*\}`)
var sassParameterName = regexp.MustCompile(`^[\t ]*\$([a-zA-Z_][a-zA-Z0-9_-]*)`)
var fashionJavaScriptFunctionCall = regexp.MustCompile(`(?i)\b(map_merge|intersect[-_]arguments)[\t\r\n ]*\(`)
var fashionNullSafeBoxRotationAssignment = regexp.MustCompile(`(?i)^(?:rotate90|rotate180|rotate270)[\t ]*\([\t ]*\$([a-zA-Z_][a-zA-Z0-9_-]*)[\t ]*\)$`)
var themeBackgroundImageFunctionDefinition = regexp.MustCompile(`(?m)^[\t ]*@function[\t ]+theme-background-image[\t ]*\(`)
var fashionImageFunctionDefinition = regexp.MustCompile(`(?m)^[\t ]*@function[\t ]+(?:theme-background-image|get-image-url)[\t ]*\(`)
var themeBackgroundImageCall = regexp.MustCompile(`theme-background-image[\t ]*\(`)
var themeBackgroundImageLiteralCall = regexp.MustCompile(`theme-background-image\([\t ]*(?:'([^']*)'|"([^"]*)")[\t ]*(?:,[\t ]*(?:'([^']*)'|"([^"]*)"))?(?:,[\t ]*(?:true|false))?[\t ]*\)`)
var fashionResourcePathAssignment = regexp.MustCompile(`(?m)^[\t ]*\$[a-zA-Z_][a-zA-Z0-9_-]*[\t ]*:[\t ]*get-resource-path[\t ]*\(`)
var fashionBoxFunctionArgs = regexp.MustCompile(`(?m)(@function[\t ]+(?:top|right|bottom|left|vertical|horizontal|boxmax|boxmin)[\t ]*\([\t ]*\$box[\t ]*,[\t ]*\$fallbackBox)([\t ]*\))`)
var fashionButtonColorFunctionArgs = regexp.MustCompile(`(?m)(@function[\t ]+button-(?:hovered|pressed|focused|disabled)-color[\t ]*\([^,()]+,[\t ]*)(\$color)([\t ]*\))`)
var fashionDuplicateMixinArgument = regexp.MustCompile(`(\$[a-zA-Z_][a-zA-Z0-9_-]*)[\t ]+(\$[a-zA-Z_][a-zA-Z0-9_-]*)`)
var fashionDynamicAssignment = regexp.MustCompile(`^[\t ]*\$[a-zA-Z_][a-zA-Z0-9_-]*[\t ]*:[\t ]*dynamic\(`)
var fashionDynamicSuffix = regexp.MustCompile(`^[\t ]*(?:!default[\t ]*)?;?[\t ]*(?://.*)?$`)

func resolveClassSassSources(appRoot string, sourceRoots []string, ignore []*regexp.Regexp) ([]string, error) {
	var sources []string
	for _, sourceRoot := range sourceRoots {
		root, err := projectPath(appRoot, sourceRoot)
		if err != nil {
			return nil, fmt.Errorf("sass source path: %w", err)
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
		if strings.EqualFold(filepath.Ext(sassRoot), ".scss") {
			if !isPathWithin(sassRoot, pkg.root) {
				return nil, nil, fmt.Errorf("package %q Sass source escapes package directory", pkg.manifest.Sencha.Name)
			}
			roots = append(roots, packageSassSourceRoot{SassPath: sassRoot, Package: pkg.manifest.Sencha.Name, EntryFile: true, Theme: pkg.manifest.Sencha.Type == "theme"})
			continue
		}
		if !strings.HasSuffix(sassSlash, "/sass/src") {
			warnings = append(warnings, fmt.Sprintf("skipped package Sass path %s: expected a package sass/src directory", sassRoot))
			continue
		}
		classRoot := filepath.FromSlash(strings.TrimSuffix(sassSlash, "/sass/src") + "/src")
		if !isPathWithin(sassRoot, pkg.root) || !isPathWithin(classRoot, pkg.root) {
			return nil, nil, fmt.Errorf("package %q Sass path escapes package directory", pkg.manifest.Sencha.Name)
		}
		roots = append(roots, packageSassSourceRoot{SassPath: sassRoot, ClassPath: classRoot, Package: pkg.manifest.Sencha.Name, Theme: pkg.manifest.Sencha.Type == "theme"})
	}
	return roots, warnings, nil
}

func uniquePackageSassSourceRoots(roots []packageSassSourceRoot) []packageSassSourceRoot {
	seen := make(map[string]bool, len(roots))
	unique := make([]packageSassSourceRoot, 0, len(roots))
	for _, root := range roots {
		key := filepath.Clean(root.SassPath) + "\x00" + filepath.Clean(root.ClassPath)
		if root.EntryFile {
			key += "\x00entry"
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, root)
	}
	return unique
}

func filterThemePackageSassSources(sources []string, roots []packageSassSourceRoot, loadedClassPaths []string) []string {
	loaded := make([]string, 0, len(loadedClassPaths))
	for _, path := range loadedClassPaths {
		loaded = append(loaded, "/"+strings.Trim(filepath.ToSlash(filepath.Clean(path)), "/"))
	}
	filtered := make([]string, 0, len(sources))
	for _, source := range sources {
		explicitEntry := false
		for _, root := range roots {
			if root.Theme && root.EntryFile && filepath.Clean(source) == filepath.Clean(root.SassPath) {
				explicitEntry = true
				break
			}
		}
		include := explicitEntry
		if !explicitEntry {
			include = true
			for _, root := range roots {
				if !root.Theme || root.EntryFile || !isPathWithin(source, root.SassPath) {
					continue
				}
				relative, err := filepath.Rel(root.SassPath, source)
				if err != nil {
					include = false
					break
				}
				classSuffix := "/" + strings.TrimSuffix(filepath.ToSlash(relative), filepath.Ext(relative)) + ".js"
				include = false
				for _, classPath := range loaded {
					if strings.HasSuffix(classPath, classSuffix) {
						include = true
						break
					}
				}
				break
			}
		}
		if include {
			filtered = append(filtered, source)
		}
	}
	return filtered
}

func orderPackageSassSources(sources []string, roots []packageSassSourceRoot, toolkit string) []string {
	type orderedSource struct {
		path     string
		priority int
	}
	ordered := make([]orderedSource, 0, len(sources))
	for _, source := range sources {
		priority := 4
		for _, root := range roots {
			matchesRoot := root.EntryFile && filepath.Clean(source) == filepath.Clean(root.SassPath)
			if !root.EntryFile {
				matchesRoot = isPathWithin(source, root.SassPath)
			}
			if !matchesRoot {
				continue
			}
			rootPriority := 3
			if root.Theme {
				rootPriority = 2
			} else if root.Package == "core" {
				rootPriority = 0
			} else if root.Package == toolkit {
				rootPriority = 1
			}
			if rootPriority < priority {
				priority = rootPriority
			}
		}
		ordered = append(ordered, orderedSource{path: source, priority: priority})
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].priority < ordered[j].priority
	})
	result := make([]string, 0, len(ordered))
	for _, source := range ordered {
		result = append(result, source.path)
	}
	return result
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

func normalizeFontAwesomeIconInterpolations(source []byte) []byte {
	if !bytes.Contains(source, []byte("$fa-icons")) && !bytes.Contains(source, []byte("$fa-brand-icons")) {
		return source
	}
	return fontAwesomeEscapedIconInterpolation.ReplaceAllLiteral(source, []byte("#{ $icon }"))
}

func sassImportTreeNeedsFontAwesomeNormalization(appRoot, source string) (bool, error) {
	imports, err := resolveSassImportWatchPaths(appRoot, []string{source})
	if err != nil {
		return false, err
	}
	for _, path := range append([]string{source}, imports...) {
		data, err := os.ReadFile(path)
		if err != nil {
			return false, fmt.Errorf("read Sass import %s: %w", path, err)
		}
		if (bytes.Contains(data, []byte("$fa-icons")) || bytes.Contains(data, []byte("$fa-brand-icons"))) && fontAwesomeEscapedIconInterpolation.Match(data) {
			return true, nil
		}
	}
	return false, nil
}

func resolveThemeSassHelperSources(appRoot string, roots []string, packageRoots []packageSassEtcRoot, ignore []*regexp.Regexp) ([]string, error) {
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
	for _, packageRoot := range packageRoots {
		root := packageRoot.Path
		entryRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat Sass helper root %s: %w", root, err)
		}
		if !info.IsDir() {
			root = filepath.Dir(root)
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
			if strings.EqualFold(entry.Name(), "all.scss") || !strings.EqualFold(filepath.Ext(path), ".scss") || isIgnoredProjectPath(appRoot, path, ignore) {
				return nil
			}
			absolute, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if !info.IsDir() && absolute == entryRoot {
				return nil
			}
			data, err := os.ReadFile(absolute)
			if err != nil {
				return err
			}
			if sassSourceIsSassHelper(data) {
				helpers = append(helpers, absolute)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan Sass helper roots under %s: %w", root, err)
		}
	}
	return uniquePaths(helpers), nil
}

func sassSourceIsSassHelper(source []byte) bool {
	if sassSourceHasNestedMixin(source) || (!sassFunctionDefinition.Match(source) && !sassMixinDefinition.Match(source)) {
		return false
	}
	definitions := topLevelSassVariableNames(source)
	supported := true
	walkTopLevelSassLines(source, func(line string) {
		if !supported {
			return
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			return
		}
		if sassVariableDeclaration.MatchString(line) {
			colon := strings.IndexByte(line, ':')
			if colon >= 0 {
				for _, match := range sassVariableReference.FindAllSubmatch([]byte(line[colon+1:]), -1) {
					if !definitions[string(match[1])] {
						supported = false
						return
					}
				}
			}
		}
		if strings.HasPrefix(trimmed, ".") || strings.HasPrefix(trimmed, "#") {
			supported = false
		}
	})
	return supported
}

func normalizeThemeBackgroundImages(sourcePath string, source []byte) ([]byte, bool, string) {
	if themeBackgroundImageFunctionDefinition.Match(source) {
		return source, false, ""
	}
	if !themeBackgroundImageCall.Match(source) {
		return source, false, ""
	}
	changed := false
	unsupported := false
	output := themeBackgroundImageLiteralCall.ReplaceAllFunc(source, func(call []byte) []byte {
		match := themeBackgroundImageLiteralCall.FindSubmatch(call)
		if match == nil {
			unsupported = true
			return call
		}
		path := string(match[1])
		if path == "" {
			path = string(match[2])
		}
		extension := string(match[3])
		if extension == "" {
			extension = string(match[4])
		}
		path = filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
		path = strings.TrimPrefix(path, "images/")
		if extension == "" && filepath.Ext(path) == "" {
			extension = "png"
		}
		if extension != "" && filepath.Ext(path) == "" {
			path += "." + extension
		}
		if path == "" || filepath.IsAbs(filepath.FromSlash(path)) || path == ".." || strings.HasPrefix(path, "../") {
			unsupported = true
			return call
		}
		resourcePrefix := "../../shared/images/"
		if strings.Contains(filepath.ToSlash(sourcePath), "/packages/ux/") {
			resourcePrefix = "../../shared/ux/images/"
		}
		changed = true
		return []byte(`url("` + resourcePrefix + path + `")`)
	})
	if unsupported {
		return nil, false, "theme-background-image uses an unsupported static package image path"
	}
	return output, changed, ""
}

func stripFashionThemeBackgroundImageFunction(source []byte) []byte {
	for {
		match := fashionImageFunctionDefinition.FindIndex(source)
		if match == nil {
			return source
		}
		open := bytes.IndexByte(source[match[1]:], '{')
		if open < 0 {
			return source
		}
		open += match[1]
		depth := 0
		quote := byte(0)
		escaped := false
		end := -1
		for index := open; index < len(source); index++ {
			current := source[index]
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
			if current == '{' {
				depth++
			} else if current == '}' {
				depth--
				if depth == 0 {
					end = index + 1
					break
				}
			}
		}
		if end < 0 {
			return source
		}
		if end < len(source) && source[end] == '\r' {
			end++
		}
		if end < len(source) && source[end] == '\n' {
			end++
		}
		source = append(append([]byte(nil), source[:match[0]]...), source[end:]...)
	}
}

func packageSassEtcHelperRoots(sdkRoot string, sources []string) []packageSassEtcRoot {
	if sdkRoot == "" {
		return nil
	}
	seen := make(map[string]bool)
	var roots []packageSassEtcRoot
	for _, source := range sources {
		absolute, err := filepath.Abs(source)
		if err != nil || !isPathWithin(absolute, sdkRoot) {
			continue
		}
		for directory := filepath.Dir(absolute); ; directory = filepath.Dir(directory) {
			manifest := filepath.Join(directory, "package.json")
			if _, err := os.Stat(manifest); err == nil {
				helperRoot := filepath.Join(directory, "sass", "etc")
				if info, err := os.Stat(helperRoot); err == nil && info.IsDir() && !seen[helperRoot] {
					seen[helperRoot] = true
					roots = append(roots, packageSassEtcRoot{Path: helperRoot})
				}
				break
			}
			parent := filepath.Dir(directory)
			if parent == directory || !isPathWithin(parent, sdkRoot) {
				break
			}
		}
	}
	return roots
}

func normalizeFashionBoxFunctionDefaults(source []byte) []byte {
	return fashionBoxFunctionArgs.ReplaceAll(source, []byte("${1}: null${2}"))
}

func normalizeFashionResourcePaths(source []byte) ([]byte, bool, string) {
	text := string(source)
	var output strings.Builder
	changed := false
	position := 0
	for position < len(text) {
		relativeStart := strings.Index(text[position:], "get-resource-path")
		if relativeStart < 0 {
			output.WriteString(text[position:])
			break
		}
		start := position + relativeStart
		nameEnd := start + len("get-resource-path")
		if nameEnd < len(text) && (text[nameEnd] == '-' || text[nameEnd] == '_' || text[nameEnd] >= 'a' && text[nameEnd] <= 'z' || text[nameEnd] >= 'A' && text[nameEnd] <= 'Z' || text[nameEnd] >= '0' && text[nameEnd] <= '9') {
			output.WriteString(text[position:nameEnd])
			position = nameEnd
			continue
		}
		cursor := nameEnd
		for cursor < len(text) && (text[cursor] == ' ' || text[cursor] == '\t' || text[cursor] == '\n' || text[cursor] == '\r') {
			cursor++
		}
		if cursor >= len(text) || text[cursor] != '(' {
			output.WriteString(text[position:nameEnd])
			position = nameEnd
			continue
		}
		lineStart := strings.LastIndex(text[:start], "\n") + 1
		if strings.HasPrefix(strings.TrimSpace(text[lineStart:start]), "@function ") {
			output.WriteString(text[position:nameEnd])
			position = nameEnd
			continue
		}
		close := sassMatchingParen([]byte(text), cursor)
		if close < 0 {
			return nil, false, "unterminated get-resource-path() Sass call"
		}
		arguments := splitTopLevelSassArguments(text[cursor+1 : close])
		if len(arguments) < 1 || len(arguments) > 2 {
			output.WriteString(text[position : close+1])
			position = close + 1
			continue
		}
		resourcePath, ok := sassStringLiteral(arguments[0])
		if !ok {
			output.WriteString(text[position : close+1])
			position = close + 1
			continue
		}
		if len(arguments) == 2 {
			pool, literal := sassStringLiteral(arguments[1])
			if !literal || (pool != "path" && pool != "shared") {
				output.WriteString(text[position : close+1])
				position = close + 1
				continue
			}
		}
		cleanPath := filepath.Clean(filepath.FromSlash(resourcePath))
		if resourcePath == "" || filepath.IsAbs(cleanPath) || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
			output.WriteString(text[position : close+1])
			position = close + 1
			continue
		}
		relativePath := filepath.ToSlash(filepath.Join("..", "..", "shared", cleanPath))
		output.WriteString(text[position:start])
		output.WriteString(strconv.Quote(relativePath))
		position = close + 1
		changed = true
	}
	if !changed {
		return source, false, ""
	}
	return []byte(output.String()), true, ""
}

func splitTopLevelSassArguments(arguments string) []string {
	var values []string
	start := 0
	parenDepth, bracketDepth, braceDepth := 0, 0, 0
	quote := byte(0)
	escaped := false
	for index := 0; index < len(arguments); index++ {
		current := arguments[index]
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
			parenDepth++
		case ')':
			parenDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case '{':
			braceDepth++
		case '}':
			braceDepth--
		case ',':
			if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
				values = append(values, strings.TrimSpace(arguments[start:index]))
				start = index + 1
			}
		}
	}
	if strings.TrimSpace(arguments[start:]) != "" {
		values = append(values, strings.TrimSpace(arguments[start:]))
	}
	return values
}

func sassStringLiteral(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if len(value) < 2 || (value[0] != '\'' && value[0] != '"') || value[len(value)-1] != value[0] {
		return "", false
	}
	content := value[1 : len(value)-1]
	if strings.ContainsRune(content, '\\') || strings.ContainsRune(content, rune(value[0])) {
		return "", false
	}
	return content, true
}

func normalizeFashionButtonColorFunctionDefaults(source []byte) []byte {
	return fashionButtonColorFunctionArgs.ReplaceAll(source, []byte("${1}${2}: null${3}"))
}

func normalizeFashionDuplicateMixinArguments(source []byte) []byte {
	matches := fashionDuplicateMixinArgument.FindAllSubmatchIndex(source, -1)
	if len(matches) == 0 {
		return source
	}
	var output strings.Builder
	previous := 0
	for _, match := range matches {
		first := string(source[match[2]:match[3]])
		second := string(source[match[4]:match[5]])
		prefix := source[:match[0]]
		include := bytes.LastIndex(prefix, []byte("@include"))
		close := bytes.LastIndexByte(prefix, ')')
		if first != second || include < 0 || include < close || !bytes.Contains(prefix[include:], []byte("(")) {
			continue
		}
		output.Write(source[previous:match[0]])
		output.WriteString(first)
		output.WriteString(": ")
		output.WriteString(second)
		previous = match[1]
	}
	if previous == 0 {
		return source
	}
	output.Write(source[previous:])
	return []byte(output.String())
}

func removeFashionScopedFunction(source []byte, declaration string) ([]byte, bool) {
	start := bytes.Index(source, []byte(declaration))
	if start < 0 {
		return source, false
	}
	brace := bytes.IndexByte(source[start:], '{')
	if brace < 0 {
		return source, false
	}
	brace += start
	depth := 0
	end := -1
	for index := brace; index < len(source); index++ {
		switch source[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = index + 1
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return source, false
	}
	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	lineEnd := end
	if lineEnd < len(source) && source[lineEnd] == '\n' {
		lineEnd++
	}
	return append(append([]byte(nil), source[:lineStart]...), source[lineEnd:]...), true
}

func normalizeFashionScopedFunctions(source []byte) []byte {
	var helpers [][]byte
	if normalized, found := removeFashionScopedFunction(source, "@function calcPos($margin)"); found {
		source = normalized
		source = bytes.ReplaceAll(source, []byte("calcPos(right($pane-margin))"), []byte("fashion-calc-pos(right($pane-margin), $cell-width)"))
		source = bytes.ReplaceAll(source, []byte("calcPos(left($pane-margin))"), []byte("fashion-calc-pos(left($pane-margin), $cell-width)"))
		helpers = append(helpers, []byte("@function fashion-calc-pos($margin, $cell-width) {\n    $sum: ceil($cell-width / 2) + $margin;\n    @if $sum != null {\n        @return calc(#{$sum} - 1em);\n    } @else {\n        @return 1em;\n    }\n}\n"))
	}
	if normalized, found := removeFashionScopedFunction(source, "@function calcOffset($padding)"); found {
		source = normalized
		source = bytes.ReplaceAll(source, []byte("calcOffset($cell-spacing)"), []byte("fashion-calc-offset($cell-spacing, $focused-cell-outline-offset)"))
		source = bytes.ReplaceAll(source, []byte("calcOffset($cell-spacing-big)"), []byte("fashion-calc-offset($cell-spacing-big, $focused-cell-outline-offset)"))
		helpers = append(helpers, []byte("@function fashion-calc-offset($padding, $focused-cell-outline-offset) {\n    $ret: 0;\n    $offset: $focused-cell-outline-offset;\n    @if $offset != null {\n        @if $padding == null {\n            $padding: 0;\n        }\n        $ret: (-$padding - 1) + $offset;\n    }\n    @return $ret;\n}\n"))
	}
	for index := len(helpers) - 1; index >= 0; index-- {
		source = append(helpers[index], source...)
	}
	return source
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
			definitions := topLevelSassVariableNames(data)
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
			nullAliases, _ := sassVariableNullAliases(source.data, candidateNulls)
			for name := range source.definitions {
				if !sourceNulls[name] && !nullAliases[name] {
					delete(candidateNulls, name)
				}
			}
			for name, isNull := range nullAliases {
				candidateNulls[name] = isNull
			}
			nullRotations := sassVariableNullRotationAssignments(source.data, candidateNulls)
			for name, isNull := range nullRotations {
				candidateNulls[name] = isNull
			}
			nullDarkens := sassVariableNullDarkenAssignments(source.data, candidateNulls)
			for name, isNull := range nullDarkens {
				candidateNulls[name] = isNull
			}
			if name := sassVariableNullReference(source.data, candidateNulls, nullAliases, nullRotations, nullDarkens); name != "" {
				progress = true
				warnings = append(warnings, fmt.Sprintf("skipped package Sass var source %s: references $%s whose Fashion value is null", source.path, name))
				continue
			}
			reason := unsupportedPackageSassVariableSource(appRoot, source.path, source.data, source.definitions, available)
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
			for name, isNull := range nullAliases {
				nullVariables[name] = isNull
			}
			for name, isNull := range nullRotations {
				nullVariables[name] = isNull
			}
			for name, isNull := range nullDarkens {
				nullVariables[name] = isNull
			}
		}
		if !progress {
			for _, source := range remaining {
				reason := unsupportedPackageSassVariableSource(appRoot, source.path, source.data, source.definitions, available)
				warnings = append(warnings, fmt.Sprintf("skipped package Sass var source %s: %s", source.path, reason))
			}
			break
		}
		pending = remaining
	}
	return uniquePaths(sources), uniquePaths(watchPaths), available, warnings, nil
}

func themePackageSassVariableSources(sources, themeRoots []string) []string {
	type themeSource struct {
		path      string
		rootOrder int
	}
	var themeSources []themeSource
	for _, source := range sources {
		for rootOrder, root := range themeRoots {
			if filepath.Clean(source) == filepath.Clean(root) || isPathWithin(source, root) {
				themeSources = append(themeSources, themeSource{path: source, rootOrder: rootOrder})
				break
			}
		}
	}
	sort.SliceStable(themeSources, func(i, j int) bool {
		return themeSources[i].rootOrder < themeSources[j].rootOrder
	})
	ordered := make([]string, 0, len(themeSources))
	for _, source := range themeSources {
		ordered = append(ordered, source.path)
	}
	return uniquePaths(ordered)
}

func themeSassVariableOverrides(sources []string) ([]byte, error) {
	type assignment struct {
		name       string
		data       []byte
		references []string
		order      int
	}
	var assignments []assignment
	finalAssignments := make(map[string]int)
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read theme Sass variable source %s: %w", source, err)
		}
		for _, declaration := range extractTopLevelSassVariableAssignments(data) {
			assignment := assignment{
				name:       declaration.name,
				data:       declaration.data,
				references: declaration.references,
				order:      len(assignments),
			}
			assignments = append(assignments, assignment)
			finalAssignments[assignment.name] = assignment.order
		}
	}
	if len(assignments) == 0 {
		return nil, nil
	}
	recomputed := make(map[string]bool)
	for name := range finalAssignments {
		if isThemeVariableForCascadeReplay(name) {
			recomputed[name] = true
		}
	}
	if len(recomputed) == 0 {
		return nil, nil
	}
	for changed := true; changed; {
		changed = false
		for name := range recomputed {
			index := finalAssignments[name]
			for _, dependency := range assignments[index].references {
				if !recomputed[dependency] {
					if _, exists := finalAssignments[dependency]; exists {
						recomputed[dependency] = true
						changed = true
					}
				}
			}
		}
	}
	selected := make(map[string]int, len(recomputed))
	for name := range recomputed {
		selected[name] = finalAssignments[name]
	}
	outgoing := make(map[string]map[string]bool, len(selected))
	indegree := make(map[string]int, len(selected))
	for name, index := range selected {
		for _, dependency := range assignments[index].references {
			if dependency == name || !recomputed[dependency] {
				continue
			}
			if outgoing[dependency] == nil {
				outgoing[dependency] = make(map[string]bool)
			}
			if !outgoing[dependency][name] {
				outgoing[dependency][name] = true
				indegree[name]++
			}
		}
	}
	less := func(left, right string) bool {
		return selected[left] < selected[right]
	}
	var ready []string
	for name := range selected {
		if indegree[name] == 0 {
			ready = append(ready, name)
		}
	}
	var ordered []string
	for len(ready) > 0 {
		sort.Slice(ready, func(i, j int) bool { return less(ready[i], ready[j]) })
		name := ready[0]
		ready = ready[1:]
		ordered = append(ordered, name)
		for dependent := range outgoing[name] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
	}
	if len(ordered) < len(selected) {
		included := make(map[string]bool, len(ordered))
		for _, name := range ordered {
			included[name] = true
		}
		var unresolved []string
		for name := range selected {
			if !included[name] {
				unresolved = append(unresolved, name)
			}
		}
		sort.Slice(unresolved, func(i, j int) bool { return less(unresolved[i], unresolved[j]) })
		ordered = append(ordered, unresolved...)
	}
	var overrides bytes.Buffer
	for _, name := range ordered {
		declaration := assignments[selected[name]].data
		if normalized, changed, reason := normalizeFashionDynamic(declaration); reason == "" && changed {
			declaration = normalized
		}
		overrides.Write(removeSassDefaultFlag(declaration))
		overrides.WriteByte('\n')
	}
	return overrides.Bytes(), nil
}

func isThemeVariableForCascadeReplay(name string) bool {
	for _, prefix := range []string{"grid-column-header-", "grid-row-", "grid-cell-", "grid-body-", "menu-item-active-", "button-default-", "button-small-font-family", "button-medium-font-family", "button-large-font-family", "panel-header-", "form-field-", "form-text-field-", "tab-", "toolbar-", "window-"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return name == "grid-header-background-color" || name == "grid-header-background-gradient" || name == "grid-header-border-color"
}

type sassVariableAssignment struct {
	name       string
	data       []byte
	references []string
}

func extractTopLevelSassVariableAssignments(source []byte) []sassVariableAssignment {
	var assignments []sassVariableAssignment
	lineOffset := 0
	walkSassLinesWithDepth(source, func(line string, braceDepth, parenDepth int) {
		startOfLine := lineOffset
		lineOffset += len(line) + 1
		if braceDepth != 0 || parenDepth != 0 {
			return
		}
		match := sassVariableDeclaration.FindStringSubmatchIndex(line)
		if len(match) < 4 {
			return
		}
		start := startOfLine + match[0]
		end := sassVariableAssignmentEnd(source, start)
		if end <= start {
			return
		}
		assignmentData := append([]byte(nil), source[start:end]...)
		if trimmed := bytes.TrimSpace(assignmentData); len(trimmed) > 0 && trimmed[len(trimmed)-1] != ';' {
			assignmentData = append(assignmentData, ';')
		}
		colon := bytes.IndexByte(assignmentData, ':')
		if colon < 0 {
			return
		}
		name := line[match[2]:match[3]]
		references := sassVariableReferencesInExpression(string(assignmentData[colon+1:]), 0)
		assignments = append(assignments, sassVariableAssignment{name: name, data: assignmentData, references: uniqueStrings(references)})
	})
	return assignments
}

func removeSassDefaultFlag(assignment []byte) []byte {
	semicolon := bytes.LastIndexByte(assignment, ';')
	if semicolon < 0 {
		return assignment
	}
	valueEnd := len(bytes.TrimRight(assignment[:semicolon], " \t\r\n"))
	trimmed := bytes.TrimRight(assignment[:valueEnd], " \t\r\n")
	if !bytes.HasSuffix(trimmed, []byte("!default")) {
		return assignment
	}
	valueEnd = len(trimmed) - len("!default")
	for valueEnd > 0 && (assignment[valueEnd-1] == ' ' || assignment[valueEnd-1] == '\t') {
		valueEnd--
	}
	result := append([]byte(nil), assignment[:valueEnd]...)
	result = append(result, ';')
	return result
}

func sassVariableAssignmentEnd(source []byte, start int) int {
	parenDepth, braceDepth, bracketDepth := 0, 0, 0
	quote := byte(0)
	escaped := false
	blockComment := false
	for index := start; index < len(source); index++ {
		current := source[index]
		if blockComment {
			if current == '*' && index+1 < len(source) && source[index+1] == '/' {
				blockComment = false
				index++
			}
			continue
		}
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
		if current == '/' && index+1 < len(source) {
			if source[index+1] == '/' {
				for index < len(source) && source[index] != '\n' {
					index++
				}
				if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
					return index
				}
				continue
			}
			if source[index+1] == '*' {
				blockComment = true
				index++
				continue
			}
		}
		switch current {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		case '{':
			braceDepth++
		case '}':
			braceDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case ';':
			if parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
				return index + 1
			}
		}
	}
	return len(source)
}

func sassVariableNullAssignments(source []byte) map[string]bool {
	assignments := make(map[string]bool)
	if normalized, _, reason := normalizeFashionDynamic(source); reason == "" {
		source = normalized
	}
	walkTopLevelSassLines(source, func(line string) {
		declaration := sassVariableDeclaration.FindStringSubmatch(line)
		if declaration == nil {
			return
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return
		}
		value := strings.TrimSpace(line[colon+1:])
		if comment := strings.Index(value, "//"); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}
		value = strings.TrimSpace(strings.TrimSuffix(value, ";"))
		assignments[string(declaration[1])] = value == "null"
	})
	return assignments
}

func sassVariableNullReference(source []byte, nullVariables, nullAliases, nullRotations, nullDarkens map[string]bool) string {
	if normalized, _, reason := normalizeFashionDynamic(source); reason == "" {
		source = normalized
	}
	missing := ""
	walkTopLevelSassLines(source, func(line string) {
		if missing != "" {
			return
		}
		declaration := sassVariableDeclaration.FindStringSubmatch(line)
		if declaration == nil {
			return
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return
		}
		variable := string(declaration[1])
		if nullAliases[variable] || nullRotations[variable] || nullDarkens[variable] {
			return
		}
		value := strings.TrimSpace(line[colon+1:])
		if comment := strings.Index(value, "//"); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}
		value = strings.TrimSpace(strings.TrimSuffix(value, ";"))
		value = strings.TrimSpace(strings.TrimSuffix(value, "!default"))
		for _, match := range sassVariableReference.FindAllSubmatch([]byte(value), -1) {
			name := string(match[1])
			if nullVariables[name] && !sassExpressionHasDirectOrOperand(value, name) {
				missing = name
				return
			}
		}
	})
	return missing
}

func sassVariableNullRotationAssignments(source []byte, nullVariables map[string]bool) map[string]bool {
	assignments := make(map[string]bool)
	if normalized, _, reason := normalizeFashionDynamic(source); reason == "" {
		source = normalized
	}
	walkTopLevelSassLines(source, func(line string) {
		declaration := sassVariableDeclaration.FindStringSubmatch(line)
		if declaration == nil {
			return
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return
		}
		value := strings.TrimSpace(line[colon+1:])
		if comment := strings.Index(value, "//"); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}
		value = strings.TrimSpace(strings.TrimSuffix(value, ";"))
		value = strings.TrimSpace(strings.TrimSuffix(value, "!default"))
		match := fashionNullSafeBoxRotationAssignment.FindStringSubmatch(value)
		if len(match) == 2 && nullVariables[match[1]] {
			assignments[string(declaration[1])] = true
		}
	})
	return assignments
}

func sassVariableNullDarkenAssignments(source []byte, nullVariables map[string]bool) map[string]bool {
	assignments := make(map[string]bool)
	if normalized, _, reason := normalizeFashionDynamic(source); reason == "" {
		source = normalized
	}
	walkTopLevelSassLines(source, func(line string) {
		declaration := sassVariableDeclaration.FindStringSubmatch(line)
		if declaration == nil {
			return
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return
		}
		value := sassCleanVariableValue(line[colon+1:])
		arguments, isDarken := sassSimpleFunctionArguments(value, "darken")
		if !isDarken || len(arguments) != 2 {
			return
		}
		color := strings.TrimSpace(arguments[0])
		if strings.HasPrefix(color, "$") && nullVariables[strings.TrimPrefix(color, "$")] {
			assignments[string(declaration[1])] = true
		}
	})
	return assignments
}

func sassCleanVariableValue(value string) string {
	if comment := strings.Index(value, "//"); comment >= 0 {
		value = value[:comment]
	}
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, ";")
	value = strings.TrimSpace(strings.TrimSuffix(value, "!default"))
	return strings.TrimSpace(value)
}

func sassSimpleFunctionArguments(expression, function string) ([]string, bool) {
	expression = strings.TrimSpace(expression)
	open := strings.IndexByte(expression, '(')
	if open <= 0 || !strings.EqualFold(strings.TrimSpace(expression[:open]), function) {
		return nil, false
	}
	close := sassMatchingParen([]byte(expression), open)
	if close != len(expression)-1 {
		return nil, false
	}
	return splitTopLevelSassArguments(expression[open+1 : close]), true
}

func normalizeFashionNullDarkenAssignments(source []byte, nullVariables map[string]bool) ([]byte, bool) {
	lines := strings.SplitAfter(string(source), "\n")
	changed := false
	braceDepth := 0
	parenDepth := 0
	for index, line := range lines {
		if braceDepth == 0 && parenDepth == 0 {
			declaration := sassVariableDeclaration.FindStringSubmatch(line)
			if declaration != nil {
				colon := strings.IndexByte(line, ':')
				if colon >= 0 {
					value := sassCleanVariableValue(line[colon+1:])
					arguments, isDarken := sassSimpleFunctionArguments(value, "darken")
					if isDarken && len(arguments) == 2 {
						color := strings.TrimSpace(arguments[0])
						if strings.HasPrefix(color, "$") && nullVariables[strings.TrimPrefix(color, "$")] {
							suffix := line[colon+1:]
							if semicolon := strings.IndexByte(suffix, ';'); semicolon >= 0 {
								suffix = suffix[semicolon:]
							} else {
								suffix = ";" + suffix
							}
							lines[index] = line[:colon+1] + " null" + suffix
							changed = true
						}
					}
				}
			}
		}
		braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
		parenDepth += strings.Count(line, "(") - strings.Count(line, ")")
		if braceDepth < 0 {
			braceDepth = 0
		}
		if parenDepth < 0 {
			parenDepth = 0
		}
	}
	if !changed {
		return source, false
	}
	return []byte(strings.Join(lines, "")), true
}

func normalizeFashionMinMaxNone(source []byte) []byte {
	matches := fashionMinMaxFunctionDefinition.FindAllSubmatchIndex(source, -1)
	if len(matches) == 0 {
		return source
	}
	var output strings.Builder
	previous := 0
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}
		openOffset := bytes.IndexByte(source[match[1]:], '{')
		if openOffset < 0 {
			continue
		}
		open := match[1] + openOffset
		end := sassBlockEnd(source, open)
		if end < 0 {
			continue
		}
		output.Write(source[previous:match[0]])
		function := string(source[match[2]:match[3]])
		block := append([]byte(nil), source[match[0]:end+1]...)
		if function == "min" {
			block = bytes.ReplaceAll(block, []byte("@if is-null($min) {"), []byte("@if is-null($min) or $min == none {"))
			block = bytes.ReplaceAll(block, []byte("@if is-null($item) {"), []byte("@if is-null($item) or $item == none {"))
			block = bytes.ReplaceAll(block, []byte("@if is-null($value1) {"), []byte("@if is-null($value1) or $value1 == none {"))
			block = bytes.ReplaceAll(block, []byte("@if is-null($value2) {"), []byte("@if is-null($value2) or $value2 == none {"))
		} else {
			block = bytes.ReplaceAll(block, []byte("@if is-null($max) {"), []byte("@if is-null($max) or $max == none {"))
			block = bytes.ReplaceAll(block, []byte("@if is-null($item) {"), []byte("@if is-null($item) or $item == none {"))
			block = bytes.ReplaceAll(block, []byte("@if is-null($value1) {"), []byte("@if is-null($value1) or $value1 == none {"))
			block = bytes.ReplaceAll(block, []byte("@if is-null($value2) {"), []byte("@if is-null($value2) or $value2 == none {"))
		}
		output.Write(block)
		previous = end + 1
	}
	if previous == 0 {
		return source
	}
	output.Write(source[previous:])
	return []byte(output.String())
}

func sassExpressionHasDirectOrOperand(expression, variable string) bool {
	expression = strings.TrimSpace(expression)
	for len(expression) >= 2 && expression[0] == '(' && sassMatchingParen([]byte(expression), 0) == len(expression)-1 {
		expression = strings.TrimSpace(expression[1 : len(expression)-1])
	}
	operands := splitTopLevelSassWordOperator(expression, "or")
	for _, operand := range operands {
		if strings.TrimSpace(operand) == "$"+variable {
			return true
		}
	}
	return false
}

func splitTopLevelSassWordOperator(expression, operator string) []string {
	var operands []string
	start := 0
	parenDepth := 0
	quote := byte(0)
	escaped := false
	for index := 0; index < len(expression); index++ {
		current := expression[index]
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
			parenDepth++
		case ')':
			parenDepth--
		default:
			if parenDepth != 0 || !strings.HasPrefix(expression[index:], operator) {
				continue
			}
			beforeSpace := index == 0 || expression[index-1] == ' ' || expression[index-1] == '\t' || expression[index-1] == '\n'
			after := index + len(operator)
			afterSpace := after == len(expression) || expression[after] == ' ' || expression[after] == '\t' || expression[after] == '\n'
			if beforeSpace && afterSpace {
				operands = append(operands, expression[start:index])
				start = after
				index = after - 1
			}
		}
	}
	operands = append(operands, expression[start:])
	return operands
}

func sassVariableNullAliases(source []byte, nullVariables map[string]bool) (map[string]bool, bool) {
	aliases := make(map[string]bool)
	if normalized, _, reason := normalizeFashionDynamic(source); reason == "" {
		source = normalized
	}
	valid := true
	walkTopLevelSassLines(source, func(line string) {
		if !valid {
			return
		}
		declaration := sassVariableDeclaration.FindStringSubmatch(line)
		if declaration == nil {
			return
		}
		colon := strings.IndexByte(line, ':')
		if colon < 0 {
			return
		}
		value := strings.TrimSpace(line[colon+1:])
		if comment := strings.Index(value, "//"); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}
		value = strings.TrimSpace(strings.TrimSuffix(value, ";"))
		value = strings.TrimSpace(strings.TrimSuffix(value, "!default"))
		for _, match := range sassVariableReference.FindAllSubmatch([]byte(value), -1) {
			name := string(match[1])
			if !nullVariables[name] {
				continue
			}
			if value != "$"+name {
				valid = false
				return
			}
			aliases[string(declaration[1])] = true
		}
	})
	return aliases, valid
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
		for name := range topLevelSassVariableNames(data) {
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

func sassScopedParameterNames(source []byte) map[string]bool {
	parameters := make(map[string]bool)
	for _, definition := range sassScopedCallableDefinition.FindAllStringIndex(string(source), -1) {
		open := definition[1] - 1
		close := sassMatchingParen(source, open)
		if close < 0 {
			continue
		}
		for _, parameter := range splitTopLevelSassArguments(string(source[open+1 : close])) {
			match := sassParameterName.FindStringSubmatch(parameter)
			if len(match) > 1 {
				parameters[match[1]] = true
			}
		}
	}
	return parameters
}

func walkTopLevelSassLines(source []byte, visit func(string)) {
	walkSassLinesWithDepth(source, func(line string, braceDepth, parenDepth int) {
		if braceDepth == 0 && parenDepth == 0 {
			visit(line)
		}
	})
}

func walkSassLinesWithDepth(source []byte, visit func(string, int, int)) {
	braceDepth := 0
	parenDepth := 0
	quote := byte(0)
	escaped := false
	blockComment := false
	for _, line := range strings.Split(string(source), "\n") {
		visit(line, braceDepth, parenDepth)
		for index := 0; index < len(line); index++ {
			current := line[index]
			if blockComment {
				if current == '*' && index+1 < len(line) && line[index+1] == '/' {
					blockComment = false
					index++
				}
				continue
			}
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
			if current == '/' && index+1 < len(line) {
				if line[index+1] == '/' {
					break
				}
				if line[index+1] == '*' {
					blockComment = true
					index++
					continue
				}
			}
			if current == '\'' || current == '"' {
				quote = current
				continue
			}
			switch current {
			case '{':
				braceDepth++
			case '}':
				braceDepth--
			case '(':
				parenDepth++
			case ')':
				parenDepth--
			}
			if braceDepth < 0 {
				braceDepth = 0
			}
			if parenDepth < 0 {
				parenDepth = 0
			}
		}
	}
}

func topLevelSassVariableNames(source []byte) map[string]bool {
	variables := make(map[string]bool)
	walkTopLevelSassLines(source, func(line string) {
		for _, match := range sassVariableDeclaration.FindAllSubmatch([]byte(line), -1) {
			variables[string(match[1])] = true
		}
	})
	return variables
}

func topLevelSassVariableReferences(source []byte) []string {
	var references []string
	assignment := false
	walkSassLinesWithDepth(source, func(line string, braceDepth, parenDepth int) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "*/") {
			return
		}
		atTopLevel := braceDepth == 0 && parenDepth == 0
		declaration := sassVariableDeclaration.FindStringSubmatch(line)
		if atTopLevel && declaration != nil {
			assignment = true
			colon := strings.IndexByte(line, ':')
			if colon >= 0 {
				references = append(references, sassVariableReferencesInExpression(line[colon+1:], parenDepth)...)
			}
		} else if assignment {
			trimmed := strings.TrimSpace(line)
			if atTopLevel && (strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, ".") || strings.HasPrefix(trimmed, "#")) {
				assignment = false
			} else {
				references = append(references, sassVariableReferencesInExpression(line, parenDepth)...)
			}
		}
		if assignment && strings.Contains(line, ";") {
			assignment = false
		}
	})
	return references
}

func sassVariableReferencesInExpression(expression string, initialParenDepth int) []string {
	var references []string
	for _, match := range sassVariableReference.FindAllStringSubmatchIndex(expression, -1) {
		if match[0] < 0 || match[1] < 0 {
			continue
		}
		if sassVariableReferenceIsNamedArgument(expression, match[0], match[1], initialParenDepth) {
			continue
		}
		references = append(references, expression[match[2]:match[3]])
	}
	return references
}

func sassVariableReferenceIsNamedArgument(expression string, start, end, initialParenDepth int) bool {
	cursor := end
	for cursor < len(expression) && (expression[cursor] == ' ' || expression[cursor] == '\t') {
		cursor++
	}
	if cursor >= len(expression) || expression[cursor] != ':' {
		return false
	}
	if initialParenDepth > 0 {
		return true
	}
	stack := make([]int, 0, 4)
	quote := byte(0)
	escaped := false
	for index := 0; index < start; index++ {
		current := expression[index]
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
			stack = append(stack, index)
		case ')':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if len(stack) == 0 {
		return false
	}
	open := stack[len(stack)-1]
	nameEnd := open
	for nameEnd > 0 && (expression[nameEnd-1] == ' ' || expression[nameEnd-1] == '\t') {
		nameEnd--
	}
	if nameEnd == 0 {
		return false
	}
	previous := expression[nameEnd-1]
	return previous == '_' || previous == '-' || previous >= 'a' && previous <= 'z' || previous >= 'A' && previous <= 'Z' || previous >= '0' && previous <= '9'
}

func unsupportedPackageSassVariableSource(appRoot, importer string, source []byte, definitions, available map[string]bool) string {
	if reason := unsupportedFashionSource(source); reason != "" {
		return reason
	}
	if bytes.Contains(source, []byte("url(")) {
		return "package Sass var uses resource URLs not covered by this step"
	}
	for _, directive := range sassImportDirective.FindAllSubmatch(source, -1) {
		if string(directive[1]) != "import" {
			return "package Sass var uses @use or @forward, which are not supported"
		}
		value := string(directive[2])
		if strings.Contains(value, "url(") {
			return "package Sass var uses CSS URL imports, which are not supported"
		}
		paths := sassImportPath.FindAllStringSubmatch(value, -1)
		if len(paths) == 0 {
			return "package Sass var contains an unsupported import"
		}
		for _, pathMatch := range paths {
			importPath := strings.TrimSpace(pathMatch[1])
			if importPath == "" || strings.HasPrefix(importPath, "sass:") || strings.Contains(importPath, "://") || strings.HasSuffix(strings.ToLower(importPath), ".css") || strings.HasSuffix(strings.ToLower(importPath), ".js") {
				return fmt.Sprintf("package Sass var import %q is not a local Sass import", importPath)
			}
			_, exists, err := resolveSassImport(appRoot, importer, importPath)
			if err != nil {
				return fmt.Sprintf("cannot resolve package Sass var import %q: %v", importPath, err)
			}
			if !exists {
				return fmt.Sprintf("references unresolved package Sass var import %q", importPath)
			}
		}
	}
	for _, name := range topLevelSassVariableReferences(source) {
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
		if root.EntryFile {
			absolute, err := filepath.Abs(root.SassPath)
			if err != nil {
				return nil, nil, nil, err
			}
			if isIgnoredProjectPath(appRoot, absolute, ignore) {
				continue
			}
			data, err := os.ReadFile(absolute)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, nil, nil, fmt.Errorf("read package Sass entry %s: %w", absolute, err)
			}
			if reason := unsupportedFashionSource(data); reason != "" {
				warnings = append(warnings, fmt.Sprintf("skipped package Sass entry %s (%s): %s", absolute, root.Package, reason))
				continue
			}
			sources = append(sources, absolute)
			watchPaths = append(watchPaths, absolute)
			continue
		}
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
			if !root.Theme {
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
			if !root.Theme {
				if reason := unsupportedPackageSassSource(data, variables); reason != "" {
					warnings = append(warnings, fmt.Sprintf("skipped package Sass source %s (%s): %s", path, root.Package, reason))
					return nil
				}
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

func sassSourcesWithMixinDefinitions(sources []string) ([]string, error) {
	var result []string
	for _, source := range uniquePaths(sources) {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read Sass mixin source %s: %w", source, err)
		}
		if sassMixinDefinition.Match(data) {
			result = append(result, source)
		}
	}
	return uniquePaths(result), nil
}

func extractSassMixinDefinitions(source []byte) []byte {
	var definitions bytes.Buffer
	lineOffset := 0
	walkSassLinesWithDepth(source, func(line string, braceDepth, _ int) {
		startOfLine := lineOffset
		lineOffset += len(line) + 1
		if braceDepth != 0 {
			return
		}
		lineBytes := []byte(line)
		mixin := sassMixinDefinition.FindIndex(lineBytes)
		function := sassFunctionDefinition.FindIndex(lineBytes)
		match := mixin
		if function != nil && (match == nil || function[0] < match[0]) {
			match = function
		}
		if match == nil {
			return
		}
		start := startOfLine + match[0]
		declarationEnd := startOfLine + match[1]
		openOffset := bytes.IndexByte(source[declarationEnd:], '{')
		if openOffset < 0 {
			return
		}
		open := declarationEnd + openOffset
		end := sassBlockEnd(source, open)
		if end < 0 {
			return
		}
		definitions.Write(source[start : end+1])
		definitions.WriteByte('\n')
	})
	return definitions.Bytes()
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
		for _, name := range sassMixinIncludesOutsideDefinitions(data) {
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

func resolveSassIncludeDefaults(sources []string, available map[string]bool) (map[string]string, error) {
	defined := make(map[string]bool, len(available))
	for name := range available {
		defined[name] = true
	}
	var references []string
	for _, source := range uniquePaths(sources) {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read Sass include source %s: %w", source, err)
		}
		for name := range sassVariableNames(data) {
			defined[name] = true
		}
		for _, match := range sassIncludeVariableReference.FindAllSubmatch(data, -1) {
			references = append(references, string(match[1]))
		}
	}
	defaults := make(map[string]string)
	for _, name := range uniqueStrings(references) {
		if !defined[name] {
			defaults[name] = "false"
		}
	}
	return defaults, nil
}

func sassMixinIncludesOutsideDefinitions(source []byte) []string {
	var includes []string
	depth := 0
	mixinDepth := 0
	pendingMixin := false
	for _, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		if mixinDepth == 0 {
			if sassMixinDefinition.MatchString(line) {
				pendingMixin = true
			} else if !pendingMixin {
				for _, match := range sassMixinInclude.FindAllSubmatch([]byte(line), -1) {
					includes = append(includes, string(match[1]))
				}
			}
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if pendingMixin && depth > 0 {
			mixinDepth = depth
			pendingMixin = false
		}
		if mixinDepth > 0 && depth < mixinDepth {
			mixinDepth = 0
		}
	}
	return uniqueStrings(includes)
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
	for name := range sassScopedParameterNames(source) {
		definitions[name] = true
	}
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

func isSassCompassCompatibilityImport(importPath string) bool {
	path := strings.TrimSuffix(strings.ToLower(filepath.ToSlash(strings.TrimSpace(importPath))), ".scss")
	switch path {
	case "compass", "compass/css3", "compass/css3/images":
		return true
	default:
		return false
	}
}

func isFashionJavaScriptImportShim(importPath string, usedFunctions map[string]bool) bool {
	base := strings.ToLower(filepath.Base(filepath.ToSlash(strings.TrimSpace(importPath))))
	switch base {
	case "font-icon.js", "extensions.js", "icon.js":
		return true
	case "functions.js":
		return !usedFunctions["map_merge"] && !usedFunctions["intersect_arguments"]
	default:
		return false
	}
}

func collectFashionJavaScriptFunctionUses(sources []string) (map[string]bool, error) {
	used := make(map[string]bool)
	for _, source := range uniquePaths(sources) {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read Sass source for Fashion function usage %s: %w", source, err)
		}
		for _, match := range fashionJavaScriptFunctionCall.FindAllSubmatch(data, -1) {
			name := strings.ReplaceAll(strings.ToLower(string(match[1])), "-", "_")
			used[name] = true
		}
	}
	return used, nil
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

func inlineSassImports(appRoot, importer string, source []byte, availableVariables map[string]bool, skipImports, usedFashionJSFunctions, nullVariables map[string]bool) ([]byte, []string, error) {
	var warnings []string
	active := make(map[string]bool)
	var expand func(string, []byte) ([]byte, error)
	expand = func(current string, data []byte) ([]byte, error) {
		data = normalizeFashionMinMaxNone(data)
		data = normalizeFontAwesomeIconInterpolations(data)
		data = stripFashionThemeBackgroundImageFunction(data)
		data, _ = normalizeFashionNullDarkenAssignments(data, nullVariables)
		data, _, _ = normalizeFashionResourcePaths(data)
		data, _ = normalizeFashionNestedMixins(data)
		data = normalizeFashionDuplicateMixinArguments(data)
		data = normalizeFashionScopedFunctions(data)
		data = normalizeFashionButtonColorFunctionDefaults(data)
		data, _, imageReason := normalizeThemeBackgroundImages(current, data)
		if imageReason != "" {
			warnings = append(warnings, fmt.Sprintf("skipped Sass source %s: %s", current, imageReason))
			return nil, nil
		}
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
			directive := text[match[0]:match[1]]
			kind := text[match[2]:match[3]]
			value := text[match[4]:match[5]]
			if braceDepth > 0 {
				if kind == "import" && strings.Contains(value, "url(") {
					output.WriteString(directive)
				} else {
					warnings = append(warnings, fmt.Sprintf("skipped Sass import nested in a block in %s", current))
				}
				previous = match[1]
				continue
			}
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
					if isFashionJavaScriptImportShim(path, usedFashionJSFunctions) {
						inlinedImport = true
						valuePrevious = pathMatch[1]
						continue
					}
					warnings = append(warnings, fmt.Sprintf("skipped Fashion JavaScript import %q in Sass source %s", path, current))
					dropDirective = true
					valuePrevious = pathMatch[1]
					continue
				}
				if kind == "import" && isSassCompassCompatibilityImport(path) {
					inlinedImport = true
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
					imported, _ = normalizeFashionNestedMixins(imported)
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
		if strings.HasSuffix(line, ";") || strings.HasSuffix(line, "}") {
			return false
		}
		return len(sassVariableDeclaration.FindAllSubmatch(source, -1)) > 0
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
		for name := range topLevelSassVariableNames(data) {
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

func normalizeFashionNestedMixins(source []byte) ([]byte, bool) {
	var helpers [][]byte
	for {
		depth := 0
		start, end := -1, -1
		lines := strings.SplitAfter(string(source), "\n")
		lineStart := 0
		for _, line := range lines {
			if depth > 0 && sassMixinDefinition.MatchString(line) {
				brace := strings.IndexByte(line, '{')
				if brace >= 0 {
					open := lineStart + brace
					close := sassBlockEnd(source, open)
					if close >= 0 {
						start, end = lineStart, close+1
					}
				}
				break
			}
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth < 0 {
				depth = 0
			}
			lineStart += len(line)
		}
		if start < 0 || end < 0 {
			break
		}
		definition := bytes.TrimSpace(source[start:end])
		captures, canHoist := fashionNestedMixinCaptures(source, definition)
		if !canHoist {
			break
		}
		nameMatch := sassMixinDefinition.FindSubmatch(definition)
		if len(nameMatch) < 2 {
			break
		}
		name := string(nameMatch[1])
		if len(captures) > 0 {
			open := bytes.IndexByte(definition, '(')
			close := sassMatchingParen(definition, open)
			if open < 0 || close < 0 {
				break
			}
			var signature strings.Builder
			signature.Write(definition[:close])
			if strings.TrimSpace(string(definition[open+1:close])) != "" {
				signature.WriteString(", ")
			}
			signature.WriteString(strings.Join(captures, ", "))
			signature.Write(definition[close:])
			definition = []byte(signature.String())
			source = appendFashionMixinCaptureArguments(source, name, captures)
		}
		helper := append(append([]byte(nil), definition...), '\n')
		helpers = append(helpers, helper)
		lineEnd := end
		if lineEnd < len(source) && source[lineEnd] == '\n' {
			lineEnd++
		}
		source = append(append([]byte(nil), source[:start]...), source[lineEnd:]...)
	}
	if len(helpers) == 0 {
		return source, false
	}
	for index := len(helpers) - 1; index >= 0; index-- {
		source = append(helpers[index], source...)
	}
	return source, true
}

func fashionNestedMixinCaptures(source, definition []byte) ([]string, bool) {
	match := sassMixinDefinition.FindSubmatch(definition)
	if len(match) < 2 {
		return nil, false
	}
	name := string(match[1])
	definedAtTopLevel := false
	walkTopLevelSassLines(source, func(line string) {
		if definition := sassMixinDefinition.FindSubmatch([]byte(line)); len(definition) > 1 && string(definition[1]) == name {
			definedAtTopLevel = true
		}
	})
	if definedAtTopLevel {
		return nil, false
	}
	open := bytes.IndexByte(definition, '(')
	close := sassMatchingParen(definition, open)
	if open < 0 || close < open {
		return nil, false
	}
	bound := make(map[string]bool)
	for _, parameter := range sassVariableReference.FindAllSubmatch(definition[open+1:close], -1) {
		bound[string(parameter[1])] = true
	}
	for _, variable := range sassVariableDeclaration.FindAllSubmatch(definition, -1) {
		bound[string(variable[1])] = true
	}
	var captures []string
	seen := make(map[string]bool)
	for _, variable := range sassVariableReference.FindAllSubmatch(definition, -1) {
		name := string(variable[1])
		if !bound[name] && !seen[name] {
			captures = append(captures, "$"+name)
			seen[name] = true
		}
	}
	return captures, true
}

func appendFashionMixinCaptureArguments(source []byte, name string, captures []string) []byte {
	needle := []byte("@include")
	var output bytes.Buffer
	previous := 0
	for search := 0; search < len(source); {
		include := bytes.Index(source[search:], needle)
		if include < 0 {
			break
		}
		include += search
		nameStart := include + len(needle)
		for nameStart < len(source) && (source[nameStart] == ' ' || source[nameStart] == '\t' || source[nameStart] == '\n' || source[nameStart] == '\r') {
			nameStart++
		}
		nameEnd := nameStart
		for nameEnd < len(source) && (source[nameEnd] == '-' || source[nameEnd] == '_' || source[nameEnd] >= 'a' && source[nameEnd] <= 'z' || source[nameEnd] >= 'A' && source[nameEnd] <= 'Z' || source[nameEnd] >= '0' && source[nameEnd] <= '9') {
			nameEnd++
		}
		if string(source[nameStart:nameEnd]) != name {
			search = nameEnd
			continue
		}
		output.Write(source[previous:nameEnd])
		arguments := strings.Join(captures, ", ")
		cursor := nameEnd
		for cursor < len(source) && (source[cursor] == ' ' || source[cursor] == '\t' || source[cursor] == '\n' || source[cursor] == '\r') {
			cursor++
		}
		if cursor < len(source) && source[cursor] == '(' {
			close := sassMatchingParen(source, cursor)
			if close < 0 {
				output.Write(source[nameEnd:])
				return output.Bytes()
			}
			output.Write(source[nameEnd : cursor+1])
			output.Write(source[cursor+1 : close])
			if strings.TrimSpace(string(source[cursor+1:close])) != "" {
				output.WriteString(", ")
			}
			output.WriteString(arguments)
			output.WriteByte(')')
			previous = close + 1
			search = previous
		} else {
			output.WriteByte('(')
			output.WriteString(arguments)
			output.WriteByte(')')
			previous = nameEnd
			search = nameEnd
		}
	}
	if previous == 0 {
		return source
	}
	output.Write(source[previous:])
	return output.Bytes()
}

func sassMatchingParen(source []byte, open int) int {
	if open < 0 || open >= len(source) || source[open] != '(' {
		return -1
	}
	depth := 0
	quote := byte(0)
	escaped := false
	for index := open; index < len(source); index++ {
		current := source[index]
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
				return index
			}
		}
	}
	return -1
}

func sassBlockEnd(source []byte, open int) int {
	depth := 0
	quote := byte(0)
	escaped := false
	blockComment := false
	for index := open; index < len(source); index++ {
		current := source[index]
		if blockComment {
			if current == '*' && index+1 < len(source) && source[index+1] == '/' {
				blockComment = false
				index++
			}
			continue
		}
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
		if current == '/' && index+1 < len(source) {
			if source[index+1] == '/' {
				for index < len(source) && source[index] != '\n' {
					index++
				}
				continue
			}
			if source[index+1] == '*' {
				blockComment = true
				index++
				continue
			}
		}
		switch current {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
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

func collectSassNullVariables(sources []string) (map[string]bool, error) {
	nullVariables := make(map[string]bool)
	for _, source := range uniquePaths(sources) {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read Sass source for null values %s: %w", source, err)
		}
		if normalized, _, reason := normalizeFashionDynamic(data); reason == "" {
			data = normalized
		}
		sourceNulls := sassVariableNullAssignments(data)
		candidateNulls := make(map[string]bool, len(nullVariables)+len(sourceNulls))
		for name, isNull := range nullVariables {
			candidateNulls[name] = isNull
		}
		for name, isNull := range sourceNulls {
			candidateNulls[name] = isNull
		}
		aliases, _ := sassVariableNullAliases(data, candidateNulls)
		for name := range topLevelSassVariableNames(data) {
			if !sourceNulls[name] && !aliases[name] {
				delete(candidateNulls, name)
			}
		}
		for name := range aliases {
			candidateNulls[name] = true
		}
		rotations := sassVariableNullRotationAssignments(data, candidateNulls)
		for name := range rotations {
			candidateNulls[name] = true
		}
		darkens := sassVariableNullDarkenAssignments(data, candidateNulls)
		for name := range darkens {
			candidateNulls[name] = true
		}
		nullVariables = candidateNulls
	}
	return nullVariables, nil
}

func compileClassSass(appRoot string, sources []string, minify bool) ([]byte, []string, error) {
	return compileClassSassWithInlineImports(appRoot, sources, minify, nil, nil, nil, nil)
}

func compileClassSassWithInlineImports(appRoot string, sources []string, minify bool, inlineImports, skipImports map[string]bool, availableVariables map[string]bool, contextVariables map[string]string) ([]byte, []string, error) {
	return compileClassSassWithInlineImportsAndMixinSources(appRoot, sources, minify, inlineImports, skipImports, availableVariables, contextVariables, nil, "", nil)
}

func compileClassSassWithInlineImportsAndMixinSources(appRoot string, sources []string, minify bool, inlineImports, skipImports map[string]bool, availableVariables map[string]bool, contextVariables map[string]string, mixinOnlySources map[string]bool, variableOverridesAfter string, variableOverrides []byte) ([]byte, []string, error) {
	if len(sources) == 0 {
		return nil, nil, nil
	}
	importedSources, err := resolveSassImportWatchPaths(appRoot, sources)
	if err != nil {
		return nil, nil, err
	}
	allSources := append(append([]string{}, sources...), importedSources...)
	inlineSources := make(map[string]bool, len(inlineImports)+len(sources))
	for source, inline := range inlineImports {
		inlineSources[source] = inline
	}
	for source := range mixinOnlySources {
		inlineSources[source] = true
	}
	for _, source := range sources {
		if inlineSources[source] {
			continue
		}
		needsNormalization, err := sassImportTreeNeedsFontAwesomeNormalization(appRoot, source)
		if err != nil {
			return nil, nil, err
		}
		if needsNormalization {
			inlineSources[source] = true
		}
	}
	usedFashionJSFunctions, err := collectFashionJavaScriptFunctionUses(allSources)
	if err != nil {
		return nil, nil, err
	}
	nullVariables, err := collectSassNullVariables(allSources)
	if err != nil {
		return nil, nil, err
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
	writeVariableOverrides := func(source string) {
		if source != variableOverridesAfter || len(variableOverrides) == 0 {
			return
		}
		if entry.Len() > 0 && !strings.HasSuffix(entry.String(), "\n") {
			entry.WriteByte('\n')
		}
		entry.Write(variableOverrides)
		if variableOverrides[len(variableOverrides)-1] != '\n' {
			entry.WriteByte('\n')
		}
		imported++
	}
	entry.WriteString("@function dynamic($value, $args...) { @return $value; }\n")
	entry.WriteString(`$current-resource-root: "resources" !default;
$relative-image-path-for-uis: "" !default;
$theme-resource-path: "" !default;
$image-search-path: "" !default;
$image-extension: "png" !default;
$slicer-image-extension: "png" !default;
$experimental-support-for-svg: false !default;
$support-for-original-webkit-gradients: false !default;
$experimental-support-for-webkit: false !default;
$experimental-support-for-mozilla: false !default;
$experimental-support-for-opera: false !default;
$experimental-support-for-pie: false !default;
$include-missing-images: false !default;
$enable-missing-image-warnings: false !default;
$output-paths: null !default;

@function map_create() { @return (); }
@function map_get($map, $key) {
    @if type-of($map) == "map" {
        @each $candidate, $value in $map {
            @if $candidate == $key { @return $value; }
        }
    }
    @return "";
}
@function map_put($map, $key, $value) {
    @if type-of($map) != "map" { $map: (); }
    @return map-merge($map, ($key: $value));
}
@function is-null($value) { @return $value == null or $value == none; }
@function file_join($first, $parts...) {
    $result: if($first == null, "", $first);
    @each $part in $parts {
        @if $part != null and $part != "" {
            @if $result == "" { $result: $part; }
            @else { $result: $result + "/" + $part; }
        }
    }
    @return $result;
}
@function theme_image_exists($search-path, $path) { @return false; }
@function theme-background-image($path, $extension: $image-extension, $nocheck: false) {
    @if $extension != null and $extension != "" { $path: $path + "." + $extension; }
    @return url(file_join("../../shared/images", $path));
}
@function parseIconArgs($glyph) {
    $character: nth($glyph, 1);
    $font-family: null;
    $rotation: null;
    $length: length($glyph);
    @if $length == 2 {
        $item: nth($glyph, 2);
        @if type-of($item) == "number" { $rotation: $item; }
        @else { $font-family: $item; }
    } @else if $length > 2 {
        $font-family: nth($glyph, 2);
        $rotation: nth($glyph, 3);
    }
    @return ($character, $font-family, $rotation);
}
@function parseFontIconArgs($glyph) {
    $character: nth($glyph, 1);
    $font-size: null;
    $font-family: null;
    $rotation: null;
    $length: length($glyph);
    @if $length == 2 {
        $item: nth($glyph, 2);
        @if type-of($item) == "number" {
            @if unit($item) == "" { $rotation: $item; }
            @else { $font-size: $item; }
        } @else { $font-family: $item; }
    } @else if $length == 3 {
        $second: nth($glyph, 2);
        $third: nth($glyph, 3);
        @if type-of($second) == "number" {
            $font-size: $second;
            @if type-of($third) == "number" { $rotation: $third; }
            @else { $font-family: $third; }
        } @else {
            $font-family: $second;
            $rotation: $third;
        }
    } @else if $length > 3 {
        $font-size: nth($glyph, 2);
        $font-family: nth($glyph, 3);
        $rotation: nth($glyph, 4);
    }
    @return ($character, $font-size, $font-family, $rotation);
}
@mixin border-radius($radius) {
    -webkit-border-radius: $radius;
    border-radius: $radius;
}
@mixin border-top-left-radius($radius) {
    -webkit-border-top-left-radius: $radius;
    border-top-left-radius: $radius;
}
@mixin border-top-right-radius($radius) {
    -webkit-border-top-right-radius: $radius;
    border-top-right-radius: $radius;
}
@mixin border-bottom-right-radius($radius) {
    -webkit-border-bottom-right-radius: $radius;
    border-bottom-right-radius: $radius;
}
@mixin border-bottom-left-radius($radius) {
    -webkit-border-bottom-left-radius: $radius;
    border-bottom-left-radius: $radius;
}
@mixin border-top-radius($radius) {
    -webkit-border-top-left-radius: $radius;
    -webkit-border-top-right-radius: $radius;
    border-top-left-radius: $radius;
    border-top-right-radius: $radius;
}
@mixin border-bottom-radius($radius) {
    -webkit-border-bottom-left-radius: $radius;
    -webkit-border-bottom-right-radius: $radius;
    border-bottom-left-radius: $radius;
    border-bottom-right-radius: $radius;
}
@mixin opacity($opacity) { opacity: $opacity; }
@mixin user-select($value) {
    -webkit-user-select: $value;
    -moz-user-select: $value;
    -ms-user-select: $value;
    user-select: $value;
}
@mixin box-shadow($shadows...) {
    -webkit-box-shadow: $shadows;
    box-shadow: $shadows;
}
@mixin rotate($angle) {
    -webkit-transform: rotate($angle);
    transform: rotate($angle);
}
@mixin scale($factor) {
    -webkit-transform: scale($factor);
    transform: scale($factor);
}
`)
	entry.WriteString(`@function parsebox($box, $index) {
    $length: length($box);
    @if $length == 0 { @return null; }
    @if $length == 1 { @return nth($box, 1); }
    @if $length == 2 {
        @if $index == 1 or $index == 3 { @return nth($box, 1); }
        @return nth($box, 2);
    }
    @if $length == 3 {
        @if $index == 1 { @return nth($box, 1); }
        @if $index == 2 or $index == 4 { @return nth($box, 2); }
        @return nth($box, 3);
    }
    @return nth($box, $index);
}
@function rotate90($box) {
    @if $box == null { @return null; }
    $length: length($box);
    @if $length == 1 { @return $box; }
    @if $length == 2 { @return nth($box, 2) nth($box, 1); }
    @return parsebox($box, 4) parsebox($box, 1) parsebox($box, 2) parsebox($box, 3);
}
@function rotate180($box) {
    @if $box == null { @return null; }
    $length: length($box);
    @if $length == 1 or $length == 2 { @return $box; }
    @return parsebox($box, 3) parsebox($box, 4) parsebox($box, 1) parsebox($box, 2);
}
@function rotate270($box) {
    @if $box == null { @return null; }
    $length: length($box);
    @if $length == 1 { @return $box; }
    @if $length == 2 { @return nth($box, 2) nth($box, 1); }
    @return parsebox($box, 2) parsebox($box, 3) parsebox($box, 4) parsebox($box, 1);
}
`)
	contextNames := make([]string, 0, len(contextVariables))
	for name := range contextVariables {
		contextNames = append(contextNames, name)
	}
	sort.Strings(contextNames)
	for _, name := range contextNames {
		entry.WriteString("$")
		entry.WriteString(name)
		entry.WriteString(": ")
		entry.WriteString(contextVariables[name])
		entry.WriteString(" !default;\n")
	}
	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, warnings, fmt.Errorf("read Sass source %s: %w", source, err)
		}
		if mixinOnlySources[source] {
			data = extractSassMixinDefinitions(data)
			if len(data) == 0 {
				continue
			}
		}
		data = normalizeFashionMinMaxNone(data)
		normalizedImages, imageChanged, imageReason := normalizeThemeBackgroundImages(source, data)
		if imageReason != "" {
			warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: %s", source, imageReason))
			continue
		}
		data = normalizedImages
		data, resourcePathChanged, resourcePathReason := normalizeFashionResourcePaths(data)
		if resourcePathReason != "" {
			warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: %s", source, resourcePathReason))
			continue
		}
		data, nestedMixinChanged := normalizeFashionNestedMixins(data)
		data = normalizeFashionBoxFunctionDefaults(data)
		if sassSourceHasNestedMixin(data) {
			warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: nested Sass mixins are unsupported", source))
			continue
		}
		normalized, dynamicChanged, reason := normalizeFashionDynamic(data)
		if reason != "" {
			if !inlineSources[source] {
				warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: %s", source, reason))
				continue
			}
			normalized = data
			dynamicChanged = false
		}
		if normalizedDarkens, darkenChanged := normalizeFashionNullDarkenAssignments(normalized, nullVariables); darkenChanged {
			normalized = normalizedDarkens
			dynamicChanged = true
		}
		if reason := unsupportedFashionSource(normalized); reason != "" {
			if !inlineSources[source] || !strings.Contains(reason, "dynamic()") {
				warnings = append(warnings, fmt.Sprintf("skipped Fashion-dependent Sass source %s: %s", source, reason))
				continue
			}
		}
		if dynamicChanged || inlineSources[source] || imageChanged || nestedMixinChanged || resourcePathChanged {
			var importWarnings []string
			normalized, importWarnings, err = inlineSassImports(appRoot, source, normalized, availableVariables, skipImports, usedFashionJSFunctions, nullVariables)
			if err != nil {
				return nil, warnings, fmt.Errorf("normalize Sass imports in %s: %w", source, err)
			}
			warnings = append(warnings, importWarnings...)
			if (inlineSources[source] || dynamicChanged) && sassSourceNeedsTrailingSemicolon(normalized) {
				normalized = append(normalized, []byte("\n;\n")...)
			}
			entry.Write(normalized)
			if len(normalized) > 0 && normalized[len(normalized)-1] != '\n' {
				entry.WriteByte('\n')
			}
			writeVariableOverrides(source)
			imported++
			continue
		}
		relative, err := filepath.Rel(appRoot, source)
		if err != nil {
			return nil, warnings, fmt.Errorf("make sass source path relative: %w", err)
		}
		entry.WriteString("@import ")
		entry.WriteString(strconv.Quote(filepath.ToSlash(relative)))
		entry.WriteString(";\n")
		writeVariableOverrides(source)
		imported++
	}
	if imported == 0 {
		return nil, warnings, nil
	}
	command := exec.Command(binary, "--no-source-map", "--style="+style, "--load-path", appRoot, "--stdin") // #nosec G702 -- SASS intentionally selects the executable; arguments are passed directly without a shell.
	command.Stdin = strings.NewReader(entry.String())
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if os.IsNotExist(err) {
			return nil, warnings, fmt.Errorf("sass executable %q was not found; install Dart Sass or set SASS to its path", binary)
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, warnings, fmt.Errorf("compile class Sass with %s: %s", binary, detail)
	}
	if start, _ := findFashionDynamicStart(stdout.String(), 0); start >= 0 {
		return nil, warnings, fmt.Errorf("compiled CSS still contains dynamic(); Sencha Fashion is required for this Sass input")
	}
	return stdout.Bytes(), warnings, nil
}

func normalizeFashionDynamic(source []byte) ([]byte, bool, string) {
	text := string(source)
	var output strings.Builder
	changed := false
	position := 0
	for position < len(text) {
		start, expressionStart := findFashionDynamicStart(text, position)
		if start < 0 {
			output.WriteString(text[position:])
			break
		}
		lineStart := strings.LastIndex(text[:start], "\n") + 1
		if !fashionDynamicAssignment.MatchString(text[lineStart:expressionStart]) {
			return nil, false, "dynamic() is supported only in simple Sass variable assignments"
		}
		closeIndex := findFashionDynamicEnd(text, expressionStart)
		if closeIndex < 0 {
			return nil, false, "unterminated dynamic() Sass variable assignment"
		}
		expression := strings.TrimSpace(text[expressionStart:closeIndex])
		expression, _, nestedReason := unwrapNestedFashionDynamic(expression)
		if nestedReason != "" {
			return nil, false, nestedReason
		}
		suffixStart := closeIndex + 1
		lineEnd := strings.IndexByte(text[suffixStart:], '\n')
		if lineEnd < 0 {
			lineEnd = len(text)
		} else {
			lineEnd += suffixStart
		}
		suffix := text[suffixStart:lineEnd]
		lineEnding := ""
		if lineEnd < len(text) {
			lineEnding = "\n"
			if strings.HasSuffix(suffix, "\r") {
				suffix = strings.TrimSuffix(suffix, "\r")
				lineEnding = "\r\n"
			}
		}
		if expression == "" || !fashionDynamicSuffix.MatchString(suffix) {
			return nil, false, "dynamic() expression is outside the supported Sass variable assignment subset"
		}
		if !strings.Contains(suffix, ";") {
			if comment := strings.Index(suffix, "//"); comment >= 0 {
				suffix = strings.TrimRight(suffix[:comment], "\t ") + "; " + suffix[comment:]
			} else {
				suffix = strings.TrimRight(suffix, "\t ") + ";"
			}
		}
		output.WriteString(text[position:start])
		output.WriteString(expression)
		output.WriteString(suffix)
		output.WriteString(lineEnding)
		changed = true
		position = lineEnd
		if lineEnd < len(text) {
			position++
		}
	}
	normalized := []byte(output.String())
	if changed {
		ordered, reordered := orderSassVariableAssignments(normalized)
		normalized = ordered
		changed = changed || reordered
	}
	return normalized, changed, ""
}

func unwrapNestedFashionDynamic(expression string) (string, bool, string) {
	var output strings.Builder
	changed := false
	position := 0
	for position < len(expression) {
		start, expressionStart := findFashionDynamicStart(expression, position)
		if start < 0 {
			output.WriteString(expression[position:])
			break
		}
		close := findFashionDynamicEnd(expression, expressionStart)
		if close < 0 {
			return "", false, "unterminated nested dynamic() Sass value"
		}
		inner, _, reason := unwrapNestedFashionDynamic(expression[expressionStart:close])
		if reason != "" {
			return "", false, reason
		}
		output.WriteString(expression[position:start])
		output.WriteString(inner)
		position = close + 1
		changed = true
	}
	return output.String(), changed, ""
}

func findFashionDynamicStart(source string, start int) (int, int) {
	quote := byte(0)
	escaped := false
	blockComment := false
	for index := start; index < len(source); index++ {
		current := source[index]
		if blockComment {
			if current == '*' && index+1 < len(source) && source[index+1] == '/' {
				blockComment = false
				index++
			}
			continue
		}
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
		if current == '/' && index+1 < len(source) {
			if source[index+1] == '/' {
				for index < len(source) && source[index] != '\n' {
					index++
				}
				continue
			}
			if source[index+1] == '*' {
				blockComment = true
				index++
				continue
			}
		}
		if current == '\'' || current == '"' {
			quote = current
			continue
		}
		if strings.HasPrefix(source[index:], "dynamic(") {
			return index, index + len("dynamic(")
		}
	}
	return -1, -1
}

func findFashionDynamicEnd(source string, start int) int {
	depth := 1
	quote := byte(0)
	escaped := false
	blockComment := false
	for index := start; index < len(source); index++ {
		current := source[index]
		if blockComment {
			if current == '*' && index+1 < len(source) && source[index+1] == '/' {
				blockComment = false
				index++
			}
			continue
		}
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
		if current == '/' && index+1 < len(source) {
			if source[index+1] == '/' {
				for index < len(source) && source[index] != '\n' {
					index++
				}
				continue
			}
			if source[index+1] == '*' {
				blockComment = true
				index++
				continue
			}
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
				return index
			}
		}
	}
	return -1
}

func orderSassVariableAssignments(source []byte) ([]byte, bool) {
	type assignmentBlock struct {
		content      string
		name         string
		dependencies map[int]bool
	}
	lines := strings.SplitAfter(string(source), "\n")
	var declarationIndexes []int
	lineIndex := 0
	valid := true
	firstTailLine := -1
	walkSassLinesWithDepth(source, func(line string, braceDepth, parenDepth int) {
		declaration := sassVariableDeclaration.MatchString(line)
		atTopLevel := braceDepth == 0 && parenDepth == 0
		trimmed := strings.TrimSpace(line)
		comment := trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "*/")
		if atTopLevel && declaration {
			if firstTailLine >= 0 {
				valid = false
			}
			declarationIndexes = append(declarationIndexes, lineIndex)
		} else if atTopLevel && !comment {
			if firstTailLine < 0 {
				firstTailLine = lineIndex
			}
		}
		lineIndex++
	})
	if !valid || len(declarationIndexes) < 2 {
		return source, false
	}
	tailStart := len(lines)
	if firstTailLine >= 0 {
		tailStart = firstTailLine
		for tailStart > 0 {
			trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(lines[tailStart-1], "\n"), "\r"))
			if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") && !strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "*/") {
				break
			}
			tailStart--
		}
	}
	starts := make([]int, len(declarationIndexes))
	for i, declarationIndex := range declarationIndexes {
		start := declarationIndex
		for start > 0 {
			trimmed := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(lines[start-1], "\n"), "\r"))
			if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") && !strings.HasPrefix(trimmed, "*") && !strings.HasPrefix(trimmed, "*/") {
				break
			}
			start--
		}
		starts[i] = start
	}
	prefixEnd := starts[0]
	prefix := strings.Join(lines[:prefixEnd], "")
	blocks := make([]assignmentBlock, len(starts))
	definitions := make(map[string]int)
	for i, start := range starts {
		end := tailStart
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		if start >= end {
			return source, false
		}
		declarationIndex := declarationIndexes[i]
		content := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(lines[declarationIndex], "\n"), "\r"))
		declaration := sassVariableDeclaration.FindStringSubmatch(content)
		if declaration == nil {
			return source, false
		}
		name := string(declaration[1])
		blocks[i] = assignmentBlock{content: strings.Join(lines[start:end], ""), name: name, dependencies: make(map[int]bool)}
		definitions[name] = i
	}
	for index := range blocks {
		for _, reference := range topLevelSassVariableReferences([]byte(blocks[index].content)) {
			if dependency, exists := definitions[reference]; exists && dependency != index {
				blocks[index].dependencies[dependency] = true
			}
		}
	}
	ordered := make([]int, 0, len(blocks))
	added := make(map[int]bool, len(blocks))
	for len(ordered) < len(blocks) {
		progress := false
		for index, block := range blocks {
			if added[index] {
				continue
			}
			ready := true
			for dependency := range block.dependencies {
				if !added[dependency] {
					ready = false
					break
				}
			}
			if ready {
				ordered = append(ordered, index)
				added[index] = true
				progress = true
				break
			}
		}
		if !progress {
			return source, false
		}
	}
	var output strings.Builder
	output.WriteString(prefix)
	changed := false
	for outputIndex, blockIndex := range ordered {
		output.WriteString(blocks[blockIndex].content)
		if outputIndex != blockIndex {
			changed = true
		}
	}
	output.WriteString(strings.Join(lines[tailStart:], ""))
	return []byte(output.String()), changed
}

func unsupportedFashionSource(source []byte) string {
	if _, _, reason := normalizeFashionDynamic(source); reason != "" {
		return reason
	}
	normalized, _, reason := normalizeFashionResourcePaths(source)
	if reason != "" {
		return reason
	}
	if fashionResourcePathAssignment.Match(normalized) {
		return "get-resource-path requires Sencha Fashion resource context"
	}
	return ""
}
