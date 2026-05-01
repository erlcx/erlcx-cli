package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/erlcx/cli/internal/auth"
)

const (
	authClientIDEnv     = "ERLCX_ROBLOX_CLIENT_ID"
	authClientSecretEnv = "ERLCX_ROBLOX_CLIENT_SECRET"
	authRedirectURIEnv  = "ERLCX_ROBLOX_REDIRECT_URI"
	authScopesEnv       = "ERLCX_ROBLOX_SCOPES"
)

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

	clientID := os.Getenv(authClientIDEnv)
	clientSecret := os.Getenv(authClientSecretEnv)
	redirectURI := os.Getenv(authRedirectURIEnv)
	scopes := os.Getenv(authScopesEnv)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	status, err := newAuthService().Login(ctx, auth.LoginOptions{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		Scopes:       scopes,
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
	clientSecret := os.Getenv(authClientSecretEnv)
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

	status, err := newAuthService().Status(ctx, auth.StatusOptions{
		Refresh:      refresh,
		ClientSecret: clientSecret,
	})
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

Environment:
  ERLCX_ROBLOX_CLIENT_ID      Roblox OAuth client ID
  ERLCX_ROBLOX_CLIENT_SECRET  Roblox OAuth client secret
  ERLCX_ROBLOX_REDIRECT_URI   Redirect URI, default http://localhost:53682/callback
  ERLCX_ROBLOX_SCOPES         Space-separated OAuth scopes
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
