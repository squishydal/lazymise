package main

import (
	"sort"

	"github.com/BurntSushi/toml"
)

type MiseConfig struct {
	Tools map[string]string `toml:"tools"`
}

type toolSource int

const (
	sourceLocal  toolSource = iota
	sourceGlobal            // read-only in the UI
)

// toolEntry is one row in the displayed list.
type toolEntry struct {
	name    string
	version string
	source  toolSource
}

type viewState int

const (
	stateList       viewState = iota
	stateAddName              // prompting for new tool name
	stateAddVersion           // prompting for new tool version
)

type model struct {
	// paths
	localPath  string // "" when no local file exists yet
	globalPath string

	// raw data (kept separate so writes only touch local)
	localTools  map[string]string
	globalTools map[string]string

	// flat sorted list for rendering / cursor
	entries []toolEntry
	cursor  int

	// input flow
	state       viewState
	inputBuffer string
	newToolName string

	// feedback
	statusMsg string
}

func initialModel() model {
	localPath := findLocalMise()
	globalPath := findGlobalMise()

	localTools := parseTools(localPath)
	globalTools := parseTools(globalPath)

	return model{
		localPath:   localPath,
		globalPath:  globalPath,
		localTools:  localTools,
		globalTools: globalTools,
		entries:     buildEntries(localTools, globalTools),
		cursor:      0,
		state:       stateList,
	}
}

func parseTools(path string) map[string]string {
	if path == "" {
		return make(map[string]string)
	}
	raw := readMiseFile(path)
	var cfg MiseConfig
	check(toml.Unmarshal(raw, &cfg))
	if cfg.Tools == nil {
		cfg.Tools = make(map[string]string)
	}
	return cfg.Tools
}

// buildEntries produces the flat list: local tools first, then global tools
// that are NOT already overridden locally.
func buildEntries(local, global map[string]string) []toolEntry {
	var entries []toolEntry

	localKeys := sortedKeys(local)
	for _, k := range localKeys {
		entries = append(entries, toolEntry{name: k, version: local[k], source: sourceLocal})
	}

	globalKeys := sortedKeys(global)
	for _, k := range globalKeys {
		if _, overridden := local[k]; !overridden {
			entries = append(entries, toolEntry{name: k, version: global[k], source: sourceGlobal})
		}
	}

	return entries
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// rebuildEntries refreshes m.entries from the current tool maps.
func (m *model) rebuildEntries() {
	m.entries = buildEntries(m.localTools, m.globalTools)
}

// saveLocal persists localTools to disk, creating the file if needed.
func (m *model) saveLocal() {
	if m.localPath == "" {
		m.localPath = ensureLocalMise()
	}
	writeLocalConfig(m.localPath, m.localTools)
}
