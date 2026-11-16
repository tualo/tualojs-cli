package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"extjs-compiler/internal/compiler"
	"extjs-compiler/internal/preview"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "extjsc:", err)
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
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	result, err := compiler.BuildWithToolkit(*appDir, *outDir, *toolkit)
	if err != nil {
		return err
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(os.Stderr, "warning:", warning)
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
		AppDir:   *appDir,
		OutDir:   *outDir,
		Toolkit:  *toolkit,
		Address:  *address,
		Interval: *interval,
	})
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  extjsc build [-app <directory>] [-out <directory>] [-toolkit classic|modern]")
	fmt.Fprintln(w, "  extjsc watch [-app <directory>] [-out <directory>] [-toolkit classic|modern] [-addr 127.0.0.1:1841]")
}
