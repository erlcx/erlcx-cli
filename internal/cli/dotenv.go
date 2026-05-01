package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")

		name, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("read %s:%d: expected NAME=value", path, lineNumber)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("read %s:%d: expected variable name", path, lineNumber)
		}

		if _, exists := os.LookupEnv(name); exists {
			continue
		}
		if err := os.Setenv(name, cleanDotEnvValue(value)); err != nil {
			return fmt.Errorf("read %s:%d: %w", path, lineNumber, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func cleanDotEnvValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		quote := value[0]
		if (quote == '"' || quote == '\'') && value[len(value)-1] == quote {
			return value[1 : len(value)-1]
		}
	}
	return value
}
