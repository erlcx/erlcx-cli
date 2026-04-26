package cli

import (
	"fmt"
	"io"
)

const (
	appName = "erlcx"
	version = "0.1.0-dev"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp(stdout)
		return 0
	case "-v", "--version", "version":
		fmt.Fprintf(stdout, "%s %s\n", appName, version)
		return 0
	default:
		fmt.Fprintf(stderr, "Unknown command: %s\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, `%s

Usage:
  erlcx <command> [options]

Commands:
  help        Show this help message
  version     Show the CLI version

`, appName)
}
