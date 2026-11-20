package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tualojs/internal/compiler"
	"tualojs/internal/preview"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tualojs:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printUsage(os.Stdout)
		return nil
	}
	switch args[0] {
	case "watch":
		return runWatch(args[1:])
	case "build":
		return runBuild(args[1:])
	default:
		return fmt.Errorf("unknown command %q (expected build or watch)", args[0])
	}
}

func runBuild(args []string) error {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	appDir := flags.String("app", ".", "Ext JS application directory containing app.json")
	outDir := flags.String("out", "build-go", "output directory for the JavaScript bundle")
	toolkit := flags.String("toolkit", "", "toolkit for universal apps (classic or modern)")
	platform := flags.String("platform", "", "platform target for Sencha conditions (desktop, phone, or tablet)")
	normalizeLetConst := flags.Bool("normalize-let-const", false, "lower direct function-scope let/const declarations to var")
	upgradeES5 := flags.Bool("upgrade-es5", false, "upgrade safe function-local ES5 var declarations to const/let")
	includeExtJS := flags.Bool("include-extjs", false, "prepend the resolved Ext JS runtime to app.js")
	jsonOutput := flags.Bool("json", false, "write the built CSS and JavaScript assets as a JSON manifest")
	basePath := flags.String("base-path", "", "public URL prefix for assets in the JSON manifest")
	mode := flags.String("mode", compiler.BuildModeDevelopment, "build mode (development or production)")
	sourceMap := flags.Bool("source-map", false, "write an app.js.map source map")
	minify := flags.Bool("minify", false, "minify JavaScript whitespace and comments")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	result, err := compiler.BuildWithBundleOptions(*appDir, *outDir, *toolkit, *platform, *normalizeLetConst, *upgradeES5, *includeExtJS, *mode, *sourceMap, *minify)
	if err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", warning)
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(compiler.NewAssetManifest(result, *basePath)); err != nil {
			return fmt.Errorf("write asset manifest: %w", err)
		}
		return nil
	}
	if result.FrameworkVersion != "" {
		framework := "Ext JS " + result.FrameworkVersion
		if len(result.Packages) > 0 {
			framework += ", packages: " + strings.Join(result.Packages, ", ")
		}
		fmt.Printf("Built %s (%s): %d source files -> %s\n", result.AppName, framework, result.SourceCount, result.OutputPath)
	} else {
		fmt.Printf("Built %s: %d source files -> %s\n", result.AppName, result.SourceCount, result.OutputPath)
	}
	return nil
}

func runWatch(args []string) error {
	flags := flag.NewFlagSet("watch", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	appDir := flags.String("app", ".", "Ext JS application directory containing app.json")
	outDir := flags.String("out", "build-go", "output directory for the JavaScript bundle")
	toolkit := flags.String("toolkit", "", "toolkit for universal apps (classic or modern)")
	platform := flags.String("platform", "", "platform target for Sencha conditions (desktop, phone, or tablet)")
	normalizeLetConst := flags.Bool("normalize-let-const", false, "lower direct function-scope let/const declarations to var")
	upgradeES5 := flags.Bool("upgrade-es5", false, "upgrade safe function-local ES5 var declarations to const/let")
	mode := flags.String("mode", compiler.BuildModeDevelopment, "build mode (development or production)")
	sourceMap := flags.Bool("source-map", false, "write an app.js.map source map")
	minify := flags.Bool("minify", false, "minify JavaScript whitespace and comments")
	address := flags.String("addr", "127.0.0.1:1841", "local HTTP listen address")
	interval := flags.Duration("interval", 500*time.Millisecond, "poll interval for source changes")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return preview.Run(ctx, preview.Config{
		AppDir:            *appDir,
		OutDir:            *outDir,
		Toolkit:           *toolkit,
		Platform:          *platform,
		NormalizeLetConst: *normalizeLetConst,
		UpgradeES5:        *upgradeES5,
		Mode:              *mode,
		SourceMap:         *sourceMap,
		Minify:            *minify,
		Address:           *address,
		Interval:          *interval,
	})
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  tualojs build [-app <directory>] [-out <directory>] [-toolkit classic|modern] [-platform desktop|phone|tablet] [-mode development|production] [-normalize-let-const] [-upgrade-es5] [-include-extjs] [-json [-base-path <url-prefix>]] [-minify] [-source-map]")
	fmt.Fprintln(w, "  tualojs watch [-app <directory>] [-out <directory>] [-toolkit classic|modern] [-platform desktop|phone|tablet] [-mode development|production] [-normalize-let-const] [-upgrade-es5] [-minify] [-source-map] [-addr 127.0.0.1:1841]")
}
