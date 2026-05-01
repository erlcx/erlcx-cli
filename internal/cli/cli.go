package cli

import (
	"fmt"
	"io"
	"strings"
)

const appName = "erlcx"

var version = "0.1.0-dev"

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if err := loadDotEnv(".env"); err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

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
		return runFileCommand("scan", args[1:], stderr)
	case "upload":
		return runFileCommand("upload", args[1:], stderr)
	case "ids":
		return runFileCommand("ids", args[1:], stderr)
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
		return runAuthLogin(args[1:], stdout, stderr)
	case "status":
		return runAuthStatus(args[1:], stdout, stderr)
	case "logout":
		return runAuthLogout(args[1:], stdout, stderr)
	case "clear-oauth-app":
		return runAuthClearOAuthApp(args[1:], stdout, stderr)
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
		return runFileCommand("lock clean", args[1:], stderr)
	default:
		fmt.Fprintf(stderr, "Unknown lock command: %s\n\n", args[0])
		printLockHelp(stderr)
		return 2
	}
}

func runFileCommand(command string, args []string, stderr io.Writer) int {
	opts, code := parseFileCommandOptions(command, args, stderr)
	if code >= 0 {
		return code
	}

	switch command {
	case "scan":
		return runScan(opts, stderr)
	case "upload":
		return runUpload(opts, stderr)
	case "ids":
		return runIDs(opts, stderr)
	case "lock clean":
		return runLockClean(opts, stderr)
	default:
		return runUnimplemented(command, stderr)
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
  clear-oauth-app
              Remove the saved custom OAuth app and current login

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
