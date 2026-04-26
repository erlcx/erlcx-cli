package cli

import (
	"fmt"
	"io"
	"strings"
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
	case "auth":
		return runAuth(args[1:], stdout, stderr)
	case "scan":
		return runUnimplemented("scan", stderr)
	case "upload":
		return runUnimplemented("upload", stderr)
	case "ids":
		return runUnimplemented("ids", stderr)
	case "lock":
		return runLock(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown command: %s\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func runAuth(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printAuthHelp(stdout)
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		printAuthHelp(stdout)
		return 0
	case "login":
		return runUnimplemented("auth login", stderr)
	case "status":
		return runUnimplemented("auth status", stderr)
	case "logout":
		return runUnimplemented("auth logout", stderr)
	default:
		fmt.Fprintf(stderr, "Unknown auth command: %s\n\n", args[0])
		printAuthHelp(stderr)
		return 2
	}
}

func runLock(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printLockHelp(stdout)
		return 0
	}

	switch args[0] {
	case "-h", "--help", "help":
		printLockHelp(stdout)
		return 0
	case "clean":
		return runUnimplemented("lock clean", stderr)
	default:
		fmt.Fprintf(stderr, "Unknown lock command: %s\n\n", args[0])
		printLockHelp(stderr)
		return 2
	}
}

func runUnimplemented(command string, stderr io.Writer) int {
	fmt.Fprintf(stderr, "%s is not implemented yet.\n", command)
	return 1
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, `%s

Usage:
  erlcx <command> [options]

Commands:
  auth        Manage Roblox login
  scan        Preview files that would be uploaded
  upload      Upload new or changed livery images
  ids         Regenerate IDs.txt from the lock file
  lock        Manage the upload lock file
  help        Show this help message
  version     Show the CLI version

`, appName)
}

func printAuthHelp(w io.Writer) {
	fmt.Fprint(w, strings.TrimLeft(`
Usage:
  erlcx auth <command>

Commands:
  login       Log in with Roblox
  status      Show the current login status
  logout      Log out and remove stored credentials

`, "\n"))
}

func printLockHelp(w io.Writer) {
	fmt.Fprint(w, strings.TrimLeft(`
Usage:
  erlcx lock <command>

Commands:
  clean       Remove stale local lock entries

`, "\n"))
}
