package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// findLocalMise returns the absolute path to ./mise.toml if it exists,
// otherwise returns "".
func findLocalMise() string {
	if _, err := os.Stat("mise.toml"); err == nil {
		abs, err := filepath.Abs("mise.toml")
		check(err)
		return abs
	}
	return ""
}

// findGlobalMise returns ~/.mise.toml, creating it if it doesn't exist.
func findGlobalMise() string {
	home, err := os.UserHomeDir()
	check(err)
	path := filepath.Join(home, ".mise.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		check(os.WriteFile(path, []byte("[tools]\n"), 0644))
	}
	return path
}

// ensureLocalMise returns the local path and creates the file if needed.
// Called right before a write so we only create the file on first actual edit.
func ensureLocalMise() string {
	abs, err := filepath.Abs("mise.toml")
	check(err)
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		check(os.WriteFile(abs, []byte("[tools]\n"), 0644))
	}
	return abs
}

func readMiseFile(path string) []byte {
	dat, err := os.ReadFile(path)
	check(err)
	return dat
}

// writeLocalConfig serialises the tools map to TOML and overwrites path.
func writeLocalConfig(path string, tools map[string]string) {
	content := "[tools]\n"
	for name, version := range tools {
		content += fmt.Sprintf("%s = \"%s\"\n", name, version)
	}
	check(os.WriteFile(path, []byte(content), 0644))
}
