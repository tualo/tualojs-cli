package compiler

import (
	"net/url"
	"path/filepath"
	"strings"
)

type ManifestAsset struct {
	URL    string `json:"url"`
	Path   string `json:"path,omitempty"`
	File   string `json:"file,omitempty"`
	Remote bool   `json:"remote,omitempty"`
}

type ManifestFile struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
	File string `json:"file"`
}

type AssetManifest struct {
	AppName       string          `json:"appName"`
	Toolkit       string          `json:"toolkit,omitempty"`
	Theme         string          `json:"theme,omitempty"`
	OutputDir     string          `json:"outputDir"`
	BasePath      string          `json:"basePath"`
	CSS           []ManifestAsset `json:"css"`
	JS            []ManifestAsset `json:"js"`
	Resources     string          `json:"resources"`
	ResourceFiles []ManifestFile  `json:"resourceFiles"`
	Files         []ManifestFile  `json:"files"`
}

// NewAssetManifest returns the CSS and JavaScript URLs emitted by a build.
// basePath is the public URL prefix corresponding to the build output root.
func NewAssetManifest(result BuildResult, basePath string) AssetManifest {
	basePath = normalizeAssetBasePath(basePath)
	outputDir := filepath.Dir(result.OutputPath)
	absoluteOutputDir, err := filepath.Abs(outputDir)
	if err == nil {
		outputDir = absoluteOutputDir
	}
	manifest := AssetManifest{
		AppName:       result.AppName,
		Toolkit:       result.Toolkit,
		Theme:         result.Theme,
		OutputDir:     outputDir,
		BasePath:      basePath,
		CSS:           make([]ManifestAsset, 0, len(result.StyleSheets)),
		JS:            make([]ManifestAsset, 0, len(result.Scripts)),
		Resources:     resolveManifestURL(basePath, "resources/", false),
		ResourceFiles: make([]ManifestFile, 0, len(result.ResourcePaths)),
		Files:         make([]ManifestFile, 0, len(result.Scripts)+len(result.StyleSheets)+len(result.ResourcePaths)+1),
	}
	seenFiles := make(map[string]bool)
	for _, stylesheet := range result.StyleSheets {
		asset := ManifestAsset{
			URL:    resolveManifestURL(basePath, stylesheet.URL, stylesheet.Remote),
			Remote: stylesheet.Remote,
		}
		if !stylesheet.Remote {
			asset.Path, asset.File = manifestFileLocation(outputDir, stylesheet.Path, stylesheet.URL)
			manifest.Files = appendUniqueManifestFile(manifest.Files, seenFiles, ManifestFile{Kind: "css", Path: asset.Path, File: asset.File})
		}
		manifest.CSS = append(manifest.CSS, asset)
	}
	for _, script := range result.Scripts {
		asset := ManifestAsset{
			URL:    resolveManifestURL(basePath, script.URL, script.Remote),
			Remote: script.Remote,
		}
		if !script.Remote {
			asset.Path, asset.File = manifestFileLocation(outputDir, filepath.Join(outputDir, filepath.FromSlash(script.URL)), script.URL)
			manifest.Files = appendUniqueManifestFile(manifest.Files, seenFiles, ManifestFile{Kind: "js", Path: asset.Path, File: asset.File})
		}
		manifest.JS = append(manifest.JS, asset)
	}
	for _, resourcePath := range result.ResourcePaths {
		path, file := manifestFileLocation(outputDir, resourcePath, "")
		resource := ManifestFile{Kind: "resource", Path: path, File: file}
		manifest.ResourceFiles = append(manifest.ResourceFiles, resource)
		manifest.Files = appendUniqueManifestFile(manifest.Files, seenFiles, resource)
	}
	if result.SourceMapPath != "" {
		path, file := manifestFileLocation(outputDir, result.SourceMapPath, "app.js.map")
		manifest.Files = appendUniqueManifestFile(manifest.Files, seenFiles, ManifestFile{Kind: "sourceMap", Path: path, File: file})
	}
	return manifest
}

func manifestFileLocation(outputDir, file, fallbackPath string) (string, string) {
	if file == "" {
		file = filepath.Join(outputDir, filepath.FromSlash(fallbackPath))
	}
	absoluteFile, err := filepath.Abs(file)
	if err == nil {
		file = absoluteFile
	}
	relative, err := filepath.Rel(outputDir, file)
	if err != nil {
		relative = fallbackPath
	}
	return filepath.ToSlash(relative), file
}

func appendUniqueManifestFile(files []ManifestFile, seen map[string]bool, file ManifestFile) []ManifestFile {
	if !seen[file.Path] {
		seen[file.Path] = true
		return append(files, file)
	}
	return files
}

func normalizeAssetBasePath(basePath string) string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		return ""
	}
	basePath = strings.TrimRight(basePath, "/")
	if basePath == "" {
		return "/"
	}
	return basePath + "/"
}

func resolveManifestURL(basePath, assetURL string, remote bool) string {
	if remote || isAbsoluteManifestURL(assetURL) || basePath == "" {
		return assetURL
	}
	if basePath == "/" {
		return "/" + strings.TrimLeft(assetURL, "/")
	}
	return basePath + strings.TrimLeft(assetURL, "/")
}

func isAbsoluteManifestURL(value string) bool {
	if strings.HasPrefix(value, "/") {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != ""
}
