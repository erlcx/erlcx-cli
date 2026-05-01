package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/erlcx/cli/internal/config"
)

type fileCommandOptions struct {
	PackDir  string
	Config   config.Config
	DryRun   bool
	FailFast bool
	Verbose  bool
}

func parseFileCommandOptions(command string, args []string, stderr io.Writer) (fileCommandOptions, int) {
	flags := flag.NewFlagSet("erlcx "+command, flag.ContinueOnError)
	flags.SetOutput(stderr)

	var templatesDir string
	var outputFile string
	var lockFile string
	var creatorType string
	var groupID int64
	var concurrency int
	var dryRun bool
	failFast := true
	var verbose bool
	var showHelp bool

	flags.BoolVar(&showHelp, "help", false, "show help")
	flags.BoolVar(&verbose, "verbose", false, "show per-file details for every scanned image")
	flags.StringVar(&templatesDir, "templates", "", "templates directory")
	flags.StringVar(&outputFile, "output", "", "output IDs file")
	flags.StringVar(&lockFile, "lock-file", "", "upload lock file")
	flags.StringVar(&creatorType, "creator", "", "creator type: user or group")
	flags.Int64Var(&groupID, "group-id", 0, "Roblox group ID")
	flags.IntVar(&concurrency, "concurrency", 0, "number of concurrent uploads")

	if command == "upload" {
		flags.BoolVar(&dryRun, "dry-run", false, "preview without uploading")
		flags.BoolVar(&failFast, "fail-fast", true, "stop after the first upload failure")
	}

	packDir, flagArgs, err := splitPackDirAndFlags(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return fileCommandOptions{}, 2
	}

	if err := flags.Parse(flagArgs); err != nil {
		return fileCommandOptions{}, 2
	}
	if showHelp {
		printFileCommandHelp(command, stderr)
		return fileCommandOptions{}, 0
	}
	if flags.NArg() != 0 || packDir == "" {
		fmt.Fprintf(stderr, "Usage: erlcx %s <pack-dir> [options]\n", command)
		return fileCommandOptions{}, 2
	}

	cfg, _, err := config.LoadForDir(packDir)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return fileCommandOptions{}, 1
	}

	if templatesDir != "" {
		cfg.TemplatesDir = resolveCLIPath(templatesDir)
	}
	if outputFile != "" {
		cfg.OutputFile = resolveCLIPath(outputFile)
	}
	if lockFile != "" {
		cfg.LockFile = resolveCLIPath(lockFile)
	}
	if creatorType != "" {
		cfg.Creator.Type = creatorType
		if creatorType == config.CreatorTypeUser {
			cfg.Creator.GroupID = nil
		}
	}
	if groupID != 0 {
		cfg.Creator.GroupID = &groupID
	}
	if concurrency != 0 {
		cfg.Concurrency = concurrency
	}

	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return fileCommandOptions{}, 2
	}

	return fileCommandOptions{
		PackDir:  packDir,
		Config:   cfg,
		DryRun:   dryRun,
		FailFast: failFast,
		Verbose:  verbose,
	}, -1
}

func resolveCLIPath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absPath
}

func splitPackDirAndFlags(args []string) (string, []string, error) {
	nonBoolFlags := map[string]bool{
		"--templates":   true,
		"--output":      true,
		"--lock-file":   true,
		"--creator":     true,
		"--group-id":    true,
		"--concurrency": true,
	}

	var packDir string
	flagArgs := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)

			name := arg
			if idx := strings.IndexByte(arg, '='); idx >= 0 {
				name = arg[:idx]
			}
			if nonBoolFlags[name] && !strings.Contains(arg, "=") {
				if i+1 >= len(args) {
					return "", nil, fmt.Errorf("missing value for %s", arg)
				}
				i++
				flagArgs = append(flagArgs, args[i])
			}
			continue
		}

		if packDir != "" {
			return "", nil, fmt.Errorf("unexpected argument: %s", arg)
		}
		packDir = arg
	}

	return packDir, flagArgs, nil
}

func printFileCommandHelp(command string, w io.Writer) {
	fmt.Fprintf(w, `Usage:
  erlcx %s <pack-dir> [options]

Options:
  --verbose               Show per-file details for every scanned image
  --templates <dir>       Templates directory
  --output <file>         Output IDs file
  --lock-file <file>      Upload lock file
  --creator <type>        Creator type: user or group
  --group-id <id>         Roblox group ID
  --concurrency <number>  Number of concurrent uploads
`, command)

	if command == "upload" {
		fmt.Fprint(w, "  --dry-run             Preview without uploading\n")
		fmt.Fprint(w, "  --fail-fast <bool>    Stop after the first upload failure (default true)\n")
	}
}
