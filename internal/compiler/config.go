package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type appConfig struct {
	Name       string                  `json:"name"`
	Namespace  string                  `json:"namespace"`
	Toolkit    string                  `json:"toolkit"`
	Theme      string                  `json:"theme"`
	Classpath  pathList                `json:"classpath"`
	Overrides  pathList                `json:"overrides"`
	Requires   pathList                `json:"requires"`
	Builds     map[string]buildProfile `json:"builds"`
	Classic    toolkitProfile          `json:"classic"`
	Modern     toolkitProfile          `json:"modern"`
	JavaScript []jsEntry               `json:"js"`
}

type buildProfile struct {
	Toolkit string `json:"toolkit"`
	Theme   string `json:"theme"`
}

type toolkitProfile struct {
	JavaScript []jsEntry `json:"js"`
}

type pathList []string

func (p *pathList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*p = []string{single}
		return nil
	}
	var multiple []string
	if err := json.Unmarshal(data, &multiple); err != nil {
		return fmt.Errorf("expected a path string or string array: %w", err)
	}
	*p = multiple
	return nil
}

type jsEntry struct {
	Path   string `json:"path"`
	Bundle bool   `json:"bundle"`
}

func loadAppConfig(appDir string) (appConfig, string, error) {
	root, err := filepath.Abs(appDir)
	if err != nil {
		return appConfig{}, "", fmt.Errorf("resolve app directory: %w", err)
	}
	configPath := filepath.Join(root, "app.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return appConfig{}, "", fmt.Errorf("read %s: %w", configPath, err)
	}

	var config appConfig
	cleaned, err := stripJSONComments(data)
	if err != nil {
		return appConfig{}, "", fmt.Errorf("parse %s: %w", configPath, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(cleaned))
	if err := decoder.Decode(&config); err != nil {
		return appConfig{}, "", fmt.Errorf("parse %s: %w", configPath, err)
	}
	if config.Name == "" {
		return appConfig{}, "", fmt.Errorf("%s: missing required field %q", configPath, "name")
	}
	if config.Namespace == "" {
		config.Namespace = config.Name
	}
	if len(config.Classpath) == 0 {
		config.Classpath = []string{"app"}
	}
	return config, root, nil
}

func stripJSONComments(data []byte) ([]byte, error) {
	cleaned := append([]byte(nil), data...)
	inString := false
	escaped := false
	for i := 0; i < len(cleaned); i++ {
		if inString {
			if escaped {
				escaped = false
			} else if cleaned[i] == '\\' {
				escaped = true
			} else if cleaned[i] == '"' {
				inString = false
			}
			continue
		}
		if cleaned[i] == '"' {
			inString = true
			continue
		}
		if cleaned[i] != '/' || i+1 >= len(cleaned) {
			continue
		}
		if cleaned[i+1] == '/' {
			for i < len(cleaned) && cleaned[i] != '\n' {
				cleaned[i] = ' '
				i++
			}
			i--
			continue
		}
		if cleaned[i+1] == '*' {
			start := i
			i += 2
			closed := false
			for i+1 < len(cleaned) {
				if cleaned[i] == '*' && cleaned[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unterminated JSON block comment")
			}
			for j := start; j < i; j++ {
				if cleaned[j] != '\n' && cleaned[j] != '\r' {
					cleaned[j] = ' '
				}
			}
			i--
		}
	}
	return cleaned, nil
}

func projectPath(root, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty project path")
	}
	absolute := path
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(root, path)
	}
	absolute, err := filepath.Abs(absolute)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil {
		return "", err
	}
	if relative == ".." || filepath.IsAbs(relative) || len(relative) > 3 && relative[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf("path %q escapes application directory", path)
	}
	return absolute, nil
}

func expandToolkitPaths(paths []string, toolkit string) ([]string, error) {
	var expanded []string
	for _, path := range paths {
		if strings.Contains(path, "${toolkit.name}") {
			if toolkit == "" {
				return nil, fmt.Errorf("path %q requires a toolkit; pass -toolkit classic or -toolkit modern", path)
			}
			path = strings.ReplaceAll(path, "${toolkit.name}", toolkit)
		}
		expanded = append(expanded, path)
	}
	return expanded, nil
}
