package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zanechee/metaprompter/internal/adapters"
	"github.com/zanechee/metaprompter/internal/config"
	"github.com/zanechee/metaprompter/internal/execute"
	"github.com/zanechee/metaprompter/internal/github"
	"github.com/zanechee/metaprompter/internal/interview"
	"github.com/zanechee/metaprompter/internal/scaffold"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "interview":
		return runInterview(ctx, args[1:])
	case "execute":
		return runExecute(ctx, args[1:])
	case "prompts":
		return runPrompts(args[1:])
	case "adapters":
		return runAdapters(args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runInterview(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("interview", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	configPath := fs.String("config", config.DefaultPath, "Path to .metaprompter config")
	dryRun := fs.Bool("dry-run", false, "Plan interview outputs without writing files or creating issues")
	framework := fs.String("framework", "", "Framework override for interview worker (codex|opencode|claude)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadRuntimeConfig(ctx, *configPath)
	if err != nil {
		return err
	}

	runner := interview.Runner{
		Config:    cfg,
		GitHub:    github.NewCLI(cfg.GitHub.Repo),
		DryRun:    *dryRun,
		Framework: *framework,
		Stdout:    os.Stdout,
	}
	return runner.Run(ctx)
}

func runExecute(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("execute", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	configPath := fs.String("config", config.DefaultPath, "Path to .metaprompter config")
	dryRun := fs.Bool("dry-run", false, "Plan execute actions without mutating git or GitHub")
	issue := fs.Int("issue", 0, "Specific issue number to execute")
	maxIssues := fs.Int("max-issues", 0, "Maximum number of issues to execute")
	childWorker := fs.Bool("child-worker", false, "Internal flag used for spawned subissue workers")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := loadRuntimeConfig(ctx, *configPath)
	if err != nil {
		return err
	}

	runner := execute.Runner{
		Config:      cfg,
		GitHub:      github.NewCLI(cfg.GitHub.Repo),
		ConfigPath:  *configPath,
		DryRun:      *dryRun,
		Issue:       *issue,
		MaxIssues:   *maxIssues,
		ChildWorker: *childWorker,
		Stdout:      os.Stdout,
	}
	return runner.Run(ctx)
}

func runPrompts(args []string) error {
	if len(args) == 0 || args[0] != "scaffold" {
		return fmt.Errorf("usage: metaprompter prompts scaffold")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := scaffold.WritePromptScaffold(cwd); err != nil {
		return err
	}
	if err := scaffold.WriteDefaultConfig(filepath.Clean(config.DefaultPath)); err != nil {
		return err
	}
	fmt.Println("Scaffolded prompts, command definitions, and default config.")
	return nil
}

func runAdapters(args []string) error {
	if len(args) == 0 || args[0] != "generate" {
		return fmt.Errorf("usage: metaprompter adapters generate [--config PATH]")
	}
	fs := flag.NewFlagSet("adapters generate", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	configPath := fs.String("config", config.DefaultPath, "Path to .metaprompter config")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	cfg, _, err := config.LoadOrDefault(*configPath)
	if err != nil {
		return err
	}
	defs, err := adapters.LoadDefinitions(cfg.Commands.SourceDir)
	if err != nil {
		return err
	}
	if err := adapters.GenerateScripts(defs, cfg.Commands.GeneratedDir); err != nil {
		return err
	}
	fmt.Printf("Generated %d adapter scripts in %s\n", len(defs), cfg.Commands.GeneratedDir)
	return nil
}

func loadRuntimeConfig(ctx context.Context, path string) (*config.Config, error) {
	cfg, _, err := config.LoadOrDefault(path)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.GitHub.Repo) == "" {
		repo, detectErr := github.DetectRepoFromGitRemote(ctx)
		if detectErr == nil {
			cfg.GitHub.Repo = repo
		}
	}
	return cfg, nil
}

func printUsage() {
	fmt.Println(`metaprompter commands:
  metaprompter interview [--dry-run] [--config PATH] [--framework NAME]
  metaprompter execute [--dry-run] [--config PATH] [--issue N] [--max-issues N]
  metaprompter prompts scaffold
  metaprompter adapters generate [--config PATH]`)
}
