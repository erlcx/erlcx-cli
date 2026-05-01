package auth

import "fmt"

type BrowserLauncher func(string) error

func OpenBrowser(url string) error {
	if err := openBrowser(url); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
