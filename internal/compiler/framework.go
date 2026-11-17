package compiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const targetExtCompatVersion = "8.0.0"

type classSourcePath struct {
	path          string
	base          string
	prefix        string
	optional      bool
	framework     bool
	alwaysInclude bool
}

type packageResolution struct {
	classpaths []classSourcePath
	overrides  []classSourcePath
	resources  []resolvedResource
	version    string
	sdkRoot    string
	packages   []string
	watchPaths []string
}

type workspaceConfig struct {
	Frameworks map[string]frameworkDescriptor `json:"frameworks"`
}

type frameworkDescriptor struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

type sdkManifest struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	CompatVersion string `json:"compatVersion"`
}

type packageManifest struct {
	Version string `json:"version"`
	Sencha  struct {
		Name          string   `json:"name"`
		Toolkit       string   `json:"toolkit"`
		AlternateName pathList `json:"alternateName"`
		Requires      pathList `json:"requires"`
		Classpath     pathList `json:"classpath"`
		Overrides     pathList `json:"overrides"`
		Resource      struct {
			Paths pathList `json:"paths"`
		} `json:"resource"`
		Resources resourceList `json:"resources"`
	} `json:"sencha"`
}

type sdkPackage struct {
	root         string
	manifestPath string
	manifest     packageManifest
}

func resolveAppPackages(appRoot string, required []string, toolkit, locale string) (packageResolution, error) {
	workspacePath := filepath.Join(appRoot, "workspace.json")
	data, err := os.ReadFile(workspacePath)
	if err != nil {
		if os.IsNotExist(err) && len(required) == 0 {
			return packageResolution{watchPaths: []string{workspacePath}}, nil
		}
		return packageResolution{}, fmt.Errorf("read workspace configuration: %w", err)
	}
	var workspace workspaceConfig
	cleaned, err := stripJSONComments(data)
	if err != nil {
		return packageResolution{}, fmt.Errorf("parse %s: %w", workspacePath, err)
	}
	if err := json.Unmarshal(cleaned, &workspace); err != nil {
		return packageResolution{}, fmt.Errorf("parse %s: %w", workspacePath, err)
	}
	framework, exists := workspace.Frameworks["ext"]
	if !exists || framework.Path == "" {
		return packageResolution{}, fmt.Errorf("workspace.json does not define frameworks.ext.path")
	}

	sdkRoot := framework.Path
	if !filepath.IsAbs(sdkRoot) {
		sdkRoot = filepath.Join(appRoot, sdkRoot)
	}
	sdkRoot, err = filepath.Abs(sdkRoot)
	if err != nil {
		return packageResolution{}, fmt.Errorf("resolve Ext JS SDK path: %w", err)
	}
	manifestPath := filepath.Join(sdkRoot, "package.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return packageResolution{}, fmt.Errorf("read Ext JS SDK manifest %s: %w", manifestPath, err)
	}
	var sdk sdkManifest
	if err := json.Unmarshal(manifestData, &sdk); err != nil {
		return packageResolution{}, fmt.Errorf("parse %s: %w", manifestPath, err)
	}
	if sdk.Name != "ext" {
		return packageResolution{}, fmt.Errorf("SDK at %s is %q, expected Ext JS framework", sdkRoot, sdk.Name)
	}
	if framework.Version != "" && sdk.Version != framework.Version {
		return packageResolution{}, fmt.Errorf("workspace Ext JS version %s does not match SDK version %s", framework.Version, sdk.Version)
	}
	if sdk.CompatVersion != targetExtCompatVersion {
		return packageResolution{}, fmt.Errorf("Ext JS SDK compatibility version is %s; this compiler targets %s", sdk.CompatVersion, targetExtCompatVersion)
	}

	resolution := packageResolution{version: sdk.Version, sdkRoot: sdkRoot, watchPaths: []string{workspacePath, manifestPath}}
	if len(required) == 0 {
		return resolution, nil
	}
	packages, err := indexSDKPackages(sdkRoot, toolkit)
	if err != nil {
		return packageResolution{}, err
	}
	state := make(map[string]uint8)
	var ordered []*sdkPackage
	var visit func(string, []string) error
	visit = func(name string, chain []string) error {
		pkg, exists := packages[name]
		if !exists {
			return fmt.Errorf("required Ext JS package %q was not found in %s", name, sdkRoot)
		}
		key := pkg.root
		if state[key] == 2 {
			return nil
		}
		if state[key] == 1 {
			return fmt.Errorf("package dependency cycle: %s -> %s", strings.Join(chain, " -> "), name)
		}
		state[key] = 1
		for _, dependency := range pkg.manifest.Sencha.Requires {
			if err := visit(dependency, append(chain, name)); err != nil {
				return err
			}
		}
		state[key] = 2
		ordered = append(ordered, pkg)
		return nil
	}
	for _, name := range required {
		if err := visit(name, nil); err != nil {
			return packageResolution{}, err
		}
	}

	seenNames := make(map[string]bool)
	for _, pkg := range ordered {
		name := pkg.manifest.Sencha.Name
		if !seenNames[name] {
			resolution.packages = append(resolution.packages, name)
			seenNames[name] = true
		}
		resolution.watchPaths = append(resolution.watchPaths, pkg.manifestPath)
		classpath, err := expandPackagePaths(pkg, pkg.manifest.Sencha.Classpath, toolkit, locale)
		if err != nil {
			return packageResolution{}, err
		}
		overrides, err := expandPackagePaths(pkg, pkg.manifest.Sencha.Overrides, toolkit, locale)
		if err != nil {
			return packageResolution{}, err
		}
		resolution.classpaths = append(resolution.classpaths, classpath...)
		resolution.overrides = append(resolution.overrides, overrides...)
		resourceEntries := append(resourceList{}, pkg.manifest.Sencha.Resources...)
		for _, path := range pkg.manifest.Sencha.Resource.Paths {
			if strings.TrimSpace(path) == "" {
				continue
			}
			resourceEntries = append(resourceEntries, resourceEntry{Path: path})
		}
		resources, resourcePaths, err := expandPackageResources(pkg, resourceEntries, toolkit, locale)
		if err != nil {
			return packageResolution{}, err
		}
		resolution.resources = append(resolution.resources, resources...)
		resolution.watchPaths = append(resolution.watchPaths, resourcePaths...)
	}
	return resolution, nil
}

func expandPackageResources(pkg *sdkPackage, entries resourceList, toolkit, locale string) ([]resolvedResource, []string, error) {
	var resources []resolvedResource
	var watchPaths []string
	for _, entry := range entries {
		if entry.Remote {
			continue
		}
		path := strings.ReplaceAll(entry.Path, "${package.dir}", pkg.root)
		path = strings.ReplaceAll(path, "${package.name}", pkg.manifest.Sencha.Name)
		path, err := expandPackageLocale(path, pkg.manifest.Sencha.Name, locale)
		if err != nil {
			return nil, nil, err
		}
		if strings.Contains(path, "${toolkit.name}") {
			if toolkit == "" {
				return nil, nil, fmt.Errorf("package %q resource path %q requires a toolkit", pkg.manifest.Sencha.Name, entry.Path)
			}
			path = strings.ReplaceAll(path, "${toolkit.name}", toolkit)
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(pkg.root, path)
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve package resource path: %w", err)
		}
		relative, err := filepath.Rel(pkg.root, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return nil, nil, fmt.Errorf("package %q resource path escapes package directory", pkg.manifest.Sencha.Name)
		}
		if _, err := os.Stat(path); err != nil {
			if strings.Contains(entry.Path, "${toolkit.name}") && os.IsNotExist(err) {
				continue
			}
			return nil, nil, fmt.Errorf("package %q resource %s: %w", pkg.manifest.Sencha.Name, path, err)
		}
		output := entry.Output
		output = strings.ReplaceAll(output, "${package.name}", pkg.manifest.Sencha.Name)
		if strings.Contains(output, "${toolkit.name}") {
			output = strings.ReplaceAll(output, "${toolkit.name}", toolkit)
		}
		if output == "" {
			output = "shared"
		}
		if err := validateResourceOutput(output); err != nil {
			return nil, nil, fmt.Errorf("package %q resource output: %w", pkg.manifest.Sencha.Name, err)
		}
		resources = append(resources, resolvedResource{source: path, output: output})
		watchPaths = append(watchPaths, path)
	}
	return resources, uniquePaths(watchPaths), nil
}

func indexSDKPackages(sdkRoot, toolkit string) (map[string]*sdkPackage, error) {
	locations := []string{toolkit, "packages"}
	if toolkit == "classic" {
		locations = append(locations, "modern")
	} else {
		locations = append(locations, "classic")
	}
	packages := make(map[string]*sdkPackage)
	for _, location := range locations {
		parent := filepath.Join(sdkRoot, location)
		entries, err := os.ReadDir(parent)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("list SDK packages in %s: %w", parent, err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			root := filepath.Join(parent, entry.Name())
			manifestData, err := os.ReadFile(filepath.Join(root, "package.json"))
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, fmt.Errorf("read package manifest in %s: %w", root, err)
			}
			var manifest packageManifest
			cleaned, err := stripJSONComments(manifestData)
			if err != nil {
				return nil, fmt.Errorf("parse package manifest in %s: %w", root, err)
			}
			if err := json.Unmarshal(cleaned, &manifest); err != nil {
				return nil, fmt.Errorf("parse package manifest in %s: %w", root, err)
			}
			if manifest.Sencha.Name == "" {
				continue
			}
			if (manifest.Sencha.Toolkit == "classic" || manifest.Sencha.Toolkit == "modern") && toolkit != "" && manifest.Sencha.Toolkit != toolkit {
				continue
			}
			pkg := &sdkPackage{root: root, manifestPath: filepath.Join(root, "package.json"), manifest: manifest}
			for _, name := range append([]string{manifest.Sencha.Name, entry.Name()}, manifest.Sencha.AlternateName...) {
				if _, exists := packages[name]; !exists {
					packages[name] = pkg
				}
			}
		}
	}
	return packages, nil
}

func expandPackagePaths(pkg *sdkPackage, paths []string, toolkit, locale string) ([]classSourcePath, error) {
	var expanded []classSourcePath
	for _, path := range paths {
		path = strings.ReplaceAll(path, "${package.dir}", pkg.root)
		path = strings.ReplaceAll(path, "${package.name}", pkg.manifest.Sencha.Name)
		path, err := expandPackageLocale(path, pkg.manifest.Sencha.Name, locale)
		if err != nil {
			return nil, err
		}
		if strings.Contains(path, "${toolkit.name}") {
			if toolkit == "" {
				return nil, fmt.Errorf("package %q path %q requires a toolkit", pkg.manifest.Sencha.Name, path)
			}
			path = strings.ReplaceAll(path, "${toolkit.name}", toolkit)
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(pkg.root, path)
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve package path: %w", err)
		}
		expanded = append(expanded, classSourcePath{path: path, base: filepath.Dir(filepath.Dir(pkg.root)), prefix: "sdk/", optional: true})
	}
	return expanded, nil
}

func expandPackageLocale(path, packageName, locale string) (string, error) {
	if strings.Contains(path, "${package.locale}") {
		if locale == "" {
			return "", fmt.Errorf("package %q path %q requires app.json locale", packageName, path)
		}
		path = strings.ReplaceAll(path, "${package.locale}", locale)
	}
	return path, nil
}

func frameworkRuntimePath(config appConfig, sdkRoot, toolkit string) (string, error) {
	if sdkRoot == "" {
		return "", nil
	}
	profile := config.Modern
	defaultFile := "ext-modern-all-debug.js"
	if toolkit == "classic" {
		profile = config.Classic
		defaultFile = "ext-all-debug.js"
	}
	for _, entry := range profile.JavaScript {
		path := strings.ReplaceAll(entry.Path, "${framework.dir}", sdkRoot)
		path = strings.ReplaceAll(path, "${toolkit.name}", toolkit)
		resolved, err := projectPath(sdkRoot, path)
		if err != nil {
			return "", fmt.Errorf("framework runtime path: %w", err)
		}
		return resolved, nil
	}
	return filepath.Join(sdkRoot, "build", defaultFile), nil
}
