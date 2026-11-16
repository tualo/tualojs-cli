package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"extjs-compiler/internal/compiler"
)

type Config struct {
	AppDir            string
	OutDir            string
	Toolkit           string
	NormalizeLetConst bool
	Mode              string
	SourceMap         bool
	Minify            bool
	Address           string
	Interval          time.Duration
}

type previewState struct {
	result         compiler.BuildResult
	frameworkJS    string
	themeResources string
	themeCSS       string
	themeCSSName   string
	buildID        string
}

func Run(ctx context.Context, config Config) error {
	if config.Address == "" {
		config.Address = "127.0.0.1:1841"
	}
	if config.Interval <= 0 {
		config.Interval = 500 * time.Millisecond
	}
	if config.OutDir == "" {
		config.OutDir = "build-go"
	}
	if config.Mode == "" {
		config.Mode = compiler.BuildModeDevelopment
	}

	result, err := compiler.BuildWithOutputOptions(config.AppDir, config.OutDir, config.Toolkit, config.NormalizeLetConst, config.Mode, config.SourceMap, config.Minify)
	if err != nil {
		return err
	}
	state, err := makePreviewState(result)
	if err != nil {
		return err
	}
	var current atomic.Pointer[previewState]
	current.Store(state)
	logWarnings(result)

	server := &http.Server{
		Handler:           newHandler(config.AppDir, &current),
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", config.Address, err)
	}
	go watch(ctx, config, &current)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Previewing %s at http://%s/ (toolkit: %s)\n", result.AppName, listener.Addr(), result.Toolkit)
	err = server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func newHandler(appDir string, current *atomic.Pointer[previewState]) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.FileServer(http.Dir(appDir)).ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, previewHTML(current.Load()))
	})
	mux.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, current.Load().result.OutputPath)
	})
	mux.HandleFunc("/app.js.map", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		path := current.Load().result.SourceMapPath
		if path == "" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, path)
	})
	mux.HandleFunc("/sources/", func(w http.ResponseWriter, r *http.Request) {
		state := current.Load()
		name := strings.TrimPrefix(r.URL.Path, "/sources/")
		if name == "" || strings.Contains(name, "..") {
			http.NotFound(w, r)
			return
		}
		file, exists := state.result.SourceFiles[name]
		if !exists {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, file)
	})
	mux.HandleFunc("/resources/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		root := current.Load().result.ResourcesPath
		if info, err := os.Stat(root); err != nil || !info.IsDir() {
			root = filepath.Join(appDir, "resources")
		}
		fileServer := http.FileServer(http.Dir(root))
		http.StripPrefix("/resources/", fileServer).ServeHTTP(w, r)
	})
	mux.HandleFunc("/sdk/ext.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, current.Load().frameworkJS)
	})
	mux.HandleFunc("/sdk/classes/", func(w http.ResponseWriter, r *http.Request) {
		state := current.Load()
		className, ok := frameworkClassFromURL(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		file, exists := state.result.FrameworkFiles[className]
		if !exists {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, file)
	})
	mux.HandleFunc("/sdk/theme/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		resources := current.Load().themeResources
		if resources == "" {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		fileServer := http.FileServer(http.Dir(resources))
		http.StripPrefix("/sdk/theme/", fileServer).ServeHTTP(w, r)
	})
	mux.HandleFunc("/__build", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		_, _ = fmt.Fprint(w, current.Load().buildID)
	})
	return mux
}

func makePreviewState(result compiler.BuildResult) (*previewState, error) {
	if result.FrameworkSDKPath == "" {
		return nil, fmt.Errorf("watch preview needs an Ext JS SDK configured in workspace.json")
	}
	frameworkJS := result.FrameworkRuntimePath
	if frameworkJS == "" {
		frameworkFile := "ext-modern-all-debug.js"
		if result.Toolkit == "classic" {
			frameworkFile = "ext-all-debug.js"
		}
		frameworkJS = filepath.Join(result.FrameworkSDKPath, "build", frameworkFile)
	}
	themeResources, themeCSS, themeCSSName, found := findGeneratedAppCSS(result)
	if !found {
		themeResources = filepath.Join(result.FrameworkSDKPath, "build", result.Toolkit, result.Theme, "resources")
		themeCSS = filepath.Join(themeResources, result.Theme+"-all.css")
		themeCSSName = result.Theme + "-all.css"
		if info, err := os.Stat(themeCSS); err != nil || !info.Mode().IsRegular() {
			themeResources = ""
			themeCSS = ""
			themeCSSName = "preview.css"
		}
	}
	for _, asset := range []string{frameworkJS} {
		info, err := os.Stat(asset)
		if err != nil {
			return nil, fmt.Errorf("preview asset %s: %w", asset, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("preview asset %s is not a regular file", asset)
		}
	}
	return &previewState{
		result:         result,
		frameworkJS:    frameworkJS,
		themeResources: themeResources,
		themeCSS:       themeCSS,
		themeCSSName:   themeCSSName,
		buildID:        time.Now().UTC().Format(time.RFC3339Nano),
	}, nil
}

func findGeneratedAppCSS(result compiler.BuildResult) (string, string, string, bool) {
	modes := deduplicatePreviewStrings([]string{result.Mode, "production", "development"})
	for _, mode := range modes {
		bases := []string{
			filepath.Join(result.AppDir, "build", mode, result.AppName, result.Toolkit, "resources"),
			filepath.Join(result.AppDir, "build", mode, result.AppName, "resources"),
		}
		for _, resources := range bases {
			css := filepath.Join(resources, result.AppName+"-all.css")
			if info, err := os.Stat(css); err == nil && info.Mode().IsRegular() {
				return resources, css, filepath.Base(css), true
			}
		}
	}
	return "", "", "", false
}

func deduplicatePreviewStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	var result []string
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func previewHTML(state *previewState) string {
	frameworkClasses, _ := json.Marshal(state.result.FrameworkClasses)
	return fmt.Sprintf(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s — Go Ext JS preview</title>
  <link rel="stylesheet" href="/sdk/theme/%s">
  <script src="/sdk/ext.js"></script>
</head>
<body>
  <script>
    (() => {
      const frameworkClasses = %s;
      const startApp = () => {
        const script = document.createElement('script');
        script.src = '/app.js';
        document.body.appendChild(script);
      };
      Ext.Loader.setConfig({ enabled: true });
      Ext.Loader.setPath('Ext', '/sdk/classes');
      if (frameworkClasses.length) {
        Ext.require(frameworkClasses, startApp);
      } else {
        startApp();
      }
    })();
  </script>
  <script>
    (() => {
      let build = %q;
      setInterval(async () => {
        try {
          const response = await fetch('/__build', { cache: 'no-store' });
          const next = await response.text();
          if (build && next !== build) location.reload();
          build = next;
        } catch (_) {}
      }, 1000);
    })();
  </script>
</body>
</html>
`, html.EscapeString(state.result.AppName), html.EscapeString(state.themeCSSName), string(frameworkClasses), state.buildID)
}

func frameworkClassFromURL(requestPath string) (string, bool) {
	if !strings.HasPrefix(requestPath, "/sdk/classes/") {
		return "", false
	}
	relative := strings.TrimPrefix(requestPath, "/sdk/classes/")
	if !strings.HasSuffix(relative, ".js") || strings.Contains(relative, "..") {
		return "", false
	}
	name := strings.TrimSuffix(strings.ReplaceAll(relative, "/", "."), ".js")
	return "Ext." + name, name != ""
}

func watch(ctx context.Context, config Config, current *atomic.Pointer[previewState]) {
	state := current.Load()
	snapshot := fileSnapshot(state.result.WatchPaths, filepath.Dir(state.result.OutputPath))
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if snapshotsEqual(snapshot, fileSnapshot(state.result.WatchPaths, filepath.Dir(state.result.OutputPath))) {
				continue
			}
			result, err := compiler.BuildWithOutputOptions(config.AppDir, config.OutDir, config.Toolkit, config.NormalizeLetConst, config.Mode, config.SourceMap, config.Minify)
			if err != nil {
				log.Printf("rebuild failed: %v", err)
				snapshot = fileSnapshot(state.result.WatchPaths, filepath.Dir(state.result.OutputPath))
				continue
			}
			next, err := makePreviewState(result)
			if err != nil {
				log.Printf("rebuild preview failed: %v", err)
				snapshot = fileSnapshot(state.result.WatchPaths, filepath.Dir(state.result.OutputPath))
				continue
			}
			logWarnings(result)
			current.Store(next)
			state = next
			snapshot = fileSnapshot(state.result.WatchPaths, filepath.Dir(state.result.OutputPath))
			log.Printf("rebuilt %s (%d source files)", result.AppName, result.SourceCount)
		}
	}
}

type fileState struct {
	size    int64
	modTime int64
	mode    fs.FileMode
}

func fileSnapshot(paths []string, excluded string) map[string]fileState {
	state := make(map[string]fileState)
	for _, path := range deduplicatePaths(paths) {
		if isWithin(path, excluded) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			state[path] = fileState{modTime: -1}
			continue
		}
		if !info.IsDir() {
			state[path] = fileState{size: info.Size(), modTime: info.ModTime().UnixNano(), mode: info.Mode()}
			continue
		}
		_ = filepath.WalkDir(path, func(filePath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if isWithin(filePath, excluded) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			fileInfo, err := entry.Info()
			if err == nil {
				state[filePath] = fileState{size: fileInfo.Size(), modTime: fileInfo.ModTime().UnixNano(), mode: fileInfo.Mode()}
			}
			return nil
		})
	}
	return state
}

func deduplicatePaths(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	var result []string
	for _, path := range paths {
		if !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	return result
}

func isWithin(path, parent string) bool {
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

func snapshotsEqual(left, right map[string]fileState) bool {
	if len(left) != len(right) {
		return false
	}
	for path, state := range left {
		if right[path] != state {
			return false
		}
	}
	return true
}

func logWarnings(result compiler.BuildResult) {
	for _, warning := range result.Warnings {
		log.Printf("warning: %s", warning)
	}
}
