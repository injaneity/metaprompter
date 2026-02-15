package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultPath = ".metaprompter/config.yaml"

type Config struct {
	GitHub     GitHubConfig
	Frameworks FrameworkConfig
	Commands   CommandsConfig
	Execute    ExecuteConfig
	Checks     ChecksConfig
}

type GitHubConfig struct {
	Repo   string
	Labels LabelConfig
}

type LabelConfig struct {
	Todo       string
	InProgress string
	Blocked    string
	Done       string
}

type FrameworkConfig struct {
	Default string
	Enabled []string
}

type CommandsConfig struct {
	SourceDir    string
	GeneratedDir string
}

type ExecuteConfig struct {
	RetryAttempts       int
	AllowSubissueFanout bool
}

type ChecksConfig struct {
	TestCmd string
	LintCmd string
}

func Default() Config {
	return defaultConfig()
}

func Load(path string) (*Config, error) {
	cfg, _, err := LoadOrDefault(path)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadOrDefault loads config from path when available; otherwise it returns defaults.
// The boolean return indicates whether a config file was found.
func LoadOrDefault(path string) (*Config, bool, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultPath
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := defaultConfig()
			resolveConfiguredPaths(path, &cfg)
			if err := cfg.validate(); err != nil {
				return nil, false, err
			}
			return &cfg, false, nil
		}
		return nil, false, fmt.Errorf("read config: %w", err)
	}

	cfg := defaultConfig()
	if err := parseYAMLSubset(string(b), &cfg); err != nil {
		return nil, false, err
	}
	if err := cfg.validate(); err != nil {
		return nil, false, err
	}
	resolveConfiguredPaths(path, &cfg)
	return &cfg, true, nil
}

func defaultConfig() Config {
	return Config{
		GitHub: GitHubConfig{
			Labels: LabelConfig{
				Todo:       "todo",
				InProgress: "in_progress",
				Blocked:    "blocked",
				Done:       "done",
			},
		},
		Frameworks: FrameworkConfig{
			Default: "codex",
			Enabled: []string{"codex", "opencode", "claude"},
		},
		Commands: CommandsConfig{
			SourceDir:    "commands",
			GeneratedDir: ".metaprompter/bin",
		},
		Execute: ExecuteConfig{
			RetryAttempts:       1,
			AllowSubissueFanout: true,
		},
	}
}

func resolvePath(root, value string) string {
	if value == "" {
		return value
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Clean(filepath.Join(root, value))
}

func resolveConfiguredPaths(configPath string, cfg *Config) {
	root := filepath.Dir(configPath)
	if filepath.Base(root) == ".metaprompter" {
		root = filepath.Dir(root)
	}
	cfg.Commands.SourceDir = resolvePath(root, cfg.Commands.SourceDir)
	cfg.Commands.GeneratedDir = resolvePath(root, cfg.Commands.GeneratedDir)
}

func (c Config) validate() error {
	if strings.TrimSpace(c.Frameworks.Default) == "" {
		return errors.New("config.frameworks.default is required")
	}
	if len(c.Frameworks.Enabled) == 0 {
		return errors.New("config.frameworks.enabled must contain at least one framework")
	}
	if !contains(c.Frameworks.Enabled, c.Frameworks.Default) {
		return fmt.Errorf("framework default %q is not in frameworks.enabled", c.Frameworks.Default)
	}
	if strings.TrimSpace(c.Commands.SourceDir) == "" {
		return errors.New("config.commands.source_dir is required")
	}
	if strings.TrimSpace(c.Commands.GeneratedDir) == "" {
		return errors.New("config.commands.generated_dir is required")
	}
	if c.Execute.RetryAttempts < 0 {
		return errors.New("config.execute.retry_attempts must be >= 0")
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func parseYAMLSubset(input string, cfg *Config) error {
	var section string
	var subsection string

	scanner := bufio.NewScanner(strings.NewReader(input))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		raw := scanner.Text()
		trimmed, indent := normalizeLine(raw)
		if trimmed == "" {
			continue
		}
		if strings.HasSuffix(trimmed, ":") {
			name := strings.TrimSuffix(trimmed, ":")
			switch indent {
			case 0:
				section = name
				subsection = ""
			case 2:
				subsection = name
				if section == "frameworks" && subsection == "enabled" {
					cfg.Frameworks.Enabled = cfg.Frameworks.Enabled[:0]
				}
			default:
				return fmt.Errorf("unsupported indentation for section at line %d", lineNo)
			}
			continue
		}

		if strings.HasPrefix(trimmed, "- ") {
			if section == "frameworks" && subsection == "enabled" {
				value := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
				value = strings.Trim(value, `"'`)
				if value != "" {
					cfg.Frameworks.Enabled = append(cfg.Frameworks.Enabled, value)
				}
				continue
			}
			return fmt.Errorf("unexpected list item at line %d", lineNo)
		}

		key, value, ok := splitKV(trimmed)
		if !ok {
			return fmt.Errorf("invalid key/value at line %d", lineNo)
		}
		value = strings.Trim(value, `"'`)

		switch section {
		case "github":
			if subsection == "labels" {
				switch key {
				case "todo":
					cfg.GitHub.Labels.Todo = value
				case "in_progress":
					cfg.GitHub.Labels.InProgress = value
				case "blocked":
					cfg.GitHub.Labels.Blocked = value
				case "done":
					cfg.GitHub.Labels.Done = value
				default:
					return fmt.Errorf("unknown github.labels key %q at line %d", key, lineNo)
				}
				continue
			}
			subsection = ""
			switch key {
			case "repo":
				cfg.GitHub.Repo = value
			default:
				return fmt.Errorf("unknown github key %q at line %d", key, lineNo)
			}
		case "frameworks":
			subsection = ""
			switch key {
			case "default":
				cfg.Frameworks.Default = value
			case "enabled":
				if strings.TrimSpace(value) != "" {
					parts := strings.Split(value, ",")
					cfg.Frameworks.Enabled = cfg.Frameworks.Enabled[:0]
					for _, p := range parts {
						p = strings.TrimSpace(strings.Trim(p, `"'[]`))
						if p != "" {
							cfg.Frameworks.Enabled = append(cfg.Frameworks.Enabled, p)
						}
					}
				}
			default:
				return fmt.Errorf("unknown frameworks key %q at line %d", key, lineNo)
			}
		case "commands":
			subsection = ""
			switch key {
			case "source_dir":
				cfg.Commands.SourceDir = value
			case "generated_dir":
				cfg.Commands.GeneratedDir = value
			default:
				return fmt.Errorf("unknown commands key %q at line %d", key, lineNo)
			}
		case "execute":
			subsection = ""
			switch key {
			case "retry_attempts":
				n, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid execute.retry_attempts at line %d", lineNo)
				}
				cfg.Execute.RetryAttempts = n
			case "allow_subissue_fanout":
				v, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid execute.allow_subissue_fanout at line %d", lineNo)
				}
				cfg.Execute.AllowSubissueFanout = v
			default:
				return fmt.Errorf("unknown execute key %q at line %d", key, lineNo)
			}
		case "checks":
			subsection = ""
			switch key {
			case "test_cmd":
				cfg.Checks.TestCmd = value
			case "lint_cmd":
				cfg.Checks.LintCmd = value
			default:
				return fmt.Errorf("unknown checks key %q at line %d", key, lineNo)
			}
		default:
			return fmt.Errorf("unknown root section %q at line %d", section, lineNo)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan config: %w", err)
	}

	if len(cfg.Frameworks.Enabled) == 0 {
		cfg.Frameworks.Enabled = []string{cfg.Frameworks.Default}
	}
	return nil
}

func normalizeLine(raw string) (string, int) {
	line := raw
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = line[:idx]
	}
	line = strings.TrimRight(line, " \t")
	if strings.TrimSpace(line) == "" {
		return "", 0
	}
	indent := len(line) - len(strings.TrimLeft(line, " "))
	return strings.TrimSpace(line), indent
}

func splitKV(line string) (key string, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}
