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
	sourceGlobal
)

type toolEntry struct {
	name    string
	version string
	source  toolSource
}

type viewState int

const (
	stateList            viewState = iota
	stateAddName
	stateAddVersion
	stateVersionPicker
	stateConfirmDownload
)

type versionsLoadedMsg struct {
	tool     string
	versions []string
	err      error
}

type installDoneMsg struct {
	tool    string
	version string
	err     error
}

type model struct {
	width  int
	height int

	localPath  string
	globalPath string

	localTools  map[string]string
	globalTools map[string]string

	entries []toolEntry
	cursor  int

	state       viewState
	inputBuffer string
	newToolName string

	pendingTool     string
	versionList     []string
	versionCursor   int
	loadingVersions bool

	pendingVersion string

	installingTools map[string]bool

	statusMsg  string
	statusIsOK bool
}

func initialModel() model {
	localPath := findLocalMise()
	globalPath := findGlobalMise()
	localTools := parseTools(localPath)
	globalTools := parseTools(globalPath)

	return model{
		width:           100,
		height:          30,
		localPath:       localPath,
		globalPath:      globalPath,
		localTools:      localTools,
		globalTools:     globalTools,
		entries:         buildEntries(localTools, globalTools),
		state:           stateList,
		installingTools: make(map[string]bool),
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

func buildEntries(local, global map[string]string) []toolEntry {
	var entries []toolEntry
	for _, k := range sortedKeys(local) {
		entries = append(entries, toolEntry{name: k, version: local[k], source: sourceLocal})
	}
	for _, k := range sortedKeys(global) {
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

func (m *model) rebuildEntries() {
	m.entries = buildEntries(m.localTools, m.globalTools)
}

func (m *model) saveLocal() {
	if m.localPath == "" {
		m.localPath = ensureLocalMise()
	}
	writeLocalConfig(m.localPath, m.localTools)
}
