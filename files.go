package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// findLocalMise returns the absolute path to ./mise.toml if it exists,
// otherwise returns "".
func findLocalMise() string {
	if _, err := os.Stat("mise.toml"); err == nil {
		abs, err := filepath.Abs("mise.toml")
		if err != nil {
			return ""
		}
		return abs
	}
	return ""
}

// findGlobalMise returns ~/.mise.toml, creating it if it doesn't exist.
func findGlobalMise() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}
	path := filepath.Join(home, ".mise.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte("[tools]\n"), 0644); err != nil {
			return "", fmt.Errorf("could not create global mise.toml: %w", err)
		}
	}
	return path, nil
}

// ensureLocalMise returns the local path and creates the file if needed.
func ensureLocalMise() (string, error) {
	abs, err := filepath.Abs("mise.toml")
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		if err := os.WriteFile(abs, []byte("[tools]\n"), 0644); err != nil {
			return "", fmt.Errorf("could not create local mise.toml: %w", err)
		}
	}
	return abs, nil
}

func readMiseFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// writeLocalConfig serialises the tools map to TOML and overwrites path.
// Uses the proper TOML encoder to handle special characters safely.
func writeLocalConfig(path string, tools map[string]string) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(MiseConfig{Tools: tools}); err != nil {
		return fmt.Errorf("could not encode TOML: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	return nil
}
