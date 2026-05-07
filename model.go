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
	stateToolBrowser     // new: browse & search all available tools from mise registry
	stateAddVersion
	stateVersionPicker
	stateConfirmDownload
	stateConfirmDelete
)

// ── Registry tool ─────────────────────────────────────────────────────────────

type registryTool struct {
	name     string // short name, e.g. "node"
	full     string // registry full name, e.g. "core:node"
	desc     string // description if available
}

// ── Tea messages ──────────────────────────────────────────────────────────────

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

type registryLoadedMsg struct {
	tools []registryTool
	err   error
}

type tickMsg struct{}        // drives the spinner animation
type clearStatusMsg struct{} // clears the status bar after a delay

// ── Spinner ───────────────────────────────────────────────────────────────────

var spinFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ── Model ─────────────────────────────────────────────────────────────────────

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
	versionCache    map[string][]string // cache ls-remote results per tool
	versionCursor   int
	loadingVersions bool

	pendingVersion string

	// tool browser (stateToolBrowser)
	registryTools   []registryTool // full list from mise registry
	filteredTools   []registryTool // filtered by searchBuffer
	browserCursor   int
	loadingRegistry bool
	searchBuffer    string // live filter text

	installingTools map[string]bool
	spinFrame       int // current spinner frame index

	statusMsg  string
	statusIsOK bool

	initErr string // non-empty if startup encountered a non-fatal error
}

func initialModel() model {
	localPath := findLocalMise()

	globalPath, err := findGlobalMise()
	initErr := ""
	if err != nil {
		initErr = err.Error()
	}

	localTools, err := parseTools(localPath)
	if err != nil && initErr == "" {
		initErr = err.Error()
	}
	globalTools, err := parseTools(globalPath)
	if err != nil && initErr == "" {
		initErr = err.Error()
	}

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
		versionCache:    make(map[string][]string),
		initErr:         initErr,
	}
}

func parseTools(path string) (map[string]string, error) {
	if path == "" {
		return make(map[string]string), nil
	}
	raw, err := readMiseFile(path)
	if err != nil {
		return make(map[string]string), err
	}
	var cfg MiseConfig
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		return make(map[string]string), err
	}
	if cfg.Tools == nil {
		cfg.Tools = make(map[string]string)
	}
	return cfg.Tools, nil
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

func (m *model) saveLocal() error {
	if m.localPath == "" {
		path, err := ensureLocalMise()
		if err != nil {
			return err
		}
		m.localPath = path
	}
	return writeLocalConfig(m.localPath, m.localTools)
}
