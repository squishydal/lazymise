package main

import (
	"github.com/BurntSushi/toml"
)

type MiseConfig struct {
	Tools map[string]string `toml:"tools"`
}

type model struct {
	counter int
	tools   map[string]string
}

func initialModel() model {
	rawFile := readFiles()
	var config MiseConfig
	err := toml.Unmarshal(rawFile, &config)
	check(err)

	return model{
		counter: 0,
		tools:   config.Tools,
	}
}
