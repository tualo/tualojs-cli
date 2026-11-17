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
