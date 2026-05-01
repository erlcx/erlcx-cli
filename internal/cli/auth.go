package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/erlcx/cli/internal/auth"
)

var authPromptInput io.Reader = os.Stdin

var newAuthService = func() auth.Service {
	return auth.Service{
		OAuth: auth.OAuthClient{
			BaseURL: os.Getenv("ERLCX_ROBLOX_OAUTH_BASE_URL"),
		},
	}
}

func runAuthLogin(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("erlcx auth login", flag.ContinueOnError)
	flags.SetOutput(stderr)

	showHelp := false
	flags.BoolVar(&showHelp, "help", false, "show help")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if showHelp {
		printAuthLoginHelp(stdout)
		return 0
	}
	if flags.NArg() != 0 {
		printAuthLoginHelp(stderr)
		return 2
	}

	app, ok := promptLoginApp(stdout, stderr, authPromptInput)
	if !ok {
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	status, err := newAuthService().Login(ctx, auth.LoginOptions{
		ClientID:     app.clientID,
		ClientSecret: app.clientSecret,
	})
	if err != nil {
		fmt.Fprintf(stderr, "auth login failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Logged in with Roblox.")
	printAuthStatus(stdout, status)
	return 0
}

func runAuthStatus(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("erlcx auth status", flag.ContinueOnError)
	flags.SetOutput(stderr)

	refresh := false
	showHelp := false
	flags.BoolVar(&refresh, "refresh", false, "refresh stored token to verify login")
	flags.BoolVar(&showHelp, "help", false, "show help")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if showHelp {
		printAuthStatusHelp(stdout)
		return 0
	}
	if flags.NArg() != 0 {
		printAuthStatusHelp(stderr)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status, err := newAuthService().Status(ctx, auth.StatusOptions{Refresh: refresh})
	if err != nil {
		fmt.Fprintf(stderr, "auth status failed: %v\n", err)
		return 1
	}
	if !status.LoggedIn {
		fmt.Fprintln(stdout, "Not logged in.")
		return 0
	}

	printAuthStatus(stdout, status)
	return 0
}

func runAuthLogout(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("erlcx auth logout", flag.ContinueOnError)
	flags.SetOutput(stderr)

	showHelp := false
	flags.BoolVar(&showHelp, "help", false, "show help")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if showHelp {
		printAuthLogoutHelp(stdout)
		return 0
	}
	if flags.NArg() != 0 {
		printAuthLogoutHelp(stderr)
		return 2
	}

	if err := newAuthService().Logout(); err != nil {
		fmt.Fprintf(stderr, "auth logout failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "Logged out.")
	return 0
}

func runAuthClearOAuthApp(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("erlcx auth clear-oauth-app", flag.ContinueOnError)
	flags.SetOutput(stderr)

	showHelp := false
	flags.BoolVar(&showHelp, "help", false, "show help")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if showHelp {
		printAuthClearOAuthAppHelp(stdout)
		return 0
	}
	if flags.NArg() != 0 {
		printAuthClearOAuthAppHelp(stderr)
		return 2
	}

	if err := newAuthService().Logout(); err != nil {
		fmt.Fprintf(stderr, "auth clear-oauth-app failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "Cleared saved OAuth app and logged out.")
	return 0
}

type loginApp struct {
	clientID     string
	clientSecret string
}

func promptLoginApp(stdout io.Writer, stderr io.Writer, input io.Reader) (loginApp, bool) {
	reader := bufio.NewReader(input)

	fmt.Fprintln(stdout, "Choose how to log in:")
	fmt.Fprintln(stdout, "  1) Custom Roblox OAuth app")
	fmt.Fprintln(stdout, "  2) ERLCX account (coming later)")
	fmt.Fprint(stdout, "Selection [1]: ")

	choice, err := readPromptLine(reader)
	if err != nil {
		fmt.Fprintf(stderr, "auth login failed: read selection: %v\n", err)
		return loginApp{}, false
	}
	if choice == "" {
		choice = "1"
	}

	switch strings.ToLower(choice) {
	case "1", "custom", "custom oauth", "custom roblox oauth app":
	case "2", "erlcx", "erlcx account":
		fmt.Fprintln(stderr, "ERLCX account login is not available yet. Use Custom Roblox OAuth app for now.")
		return loginApp{}, false
	default:
		fmt.Fprintf(stderr, "Unknown login method: %s\n", choice)
		return loginApp{}, false
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Create a Roblox OAuth app and use this redirect URL:")
	fmt.Fprintf(stdout, "  %s\n\n", auth.DefaultRedirectURI)
	fmt.Fprint(stdout, "Roblox OAuth client ID: ")
	clientID, err := readPromptLine(reader)
	if err != nil {
		fmt.Fprintf(stderr, "auth login failed: read client ID: %v\n", err)
		return loginApp{}, false
	}
	if clientID == "" {
		fmt.Fprintln(stderr, "auth login failed: Roblox OAuth client ID is required")
		return loginApp{}, false
	}

	fmt.Fprint(stdout, "Roblox OAuth client secret: ")
	clientSecret, err := readPromptLine(reader)
	if err != nil {
		fmt.Fprintf(stderr, "auth login failed: read client secret: %v\n", err)
		return loginApp{}, false
	}

	return loginApp{clientID: clientID, clientSecret: clientSecret}, true
}

func readPromptLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func printAuthStatus(w io.Writer, status auth.Status) {
	account := status.Username
	if account == "" {
		account = status.DisplayName
	}
	if account == "" {
		account = status.UserID
	}
	fmt.Fprintf(w, "Account: %s\n", account)
	if status.DisplayName != "" && status.DisplayName != account {
		fmt.Fprintf(w, "Display name: %s\n", status.DisplayName)
	}
	if status.UserID != "" {
		fmt.Fprintf(w, "User ID: %s\n", status.UserID)
	}
	if status.Scopes != "" {
		fmt.Fprintf(w, "Scopes: %s\n", status.Scopes)
	}
}

func printAuthLoginHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  erlcx auth login

The login flow can use your own Roblox OAuth app.
ERLCX account login will be added later.
`)
}

func printAuthStatusHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  erlcx auth status [--refresh]

Options:
  --refresh  Refresh stored token to verify login
`)
}

func printAuthLogoutHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  erlcx auth logout
`)
}

func printAuthClearOAuthAppHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  erlcx auth clear-oauth-app

Removes the saved custom OAuth app from this PC.
This also logs out because the app secret is needed to refresh Roblox login.
`)
}
