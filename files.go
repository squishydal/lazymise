package main

import "os"

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func readFiles() []byte {
	path := "/home/squishydal/Projects/lazymise/mise.toml"
	dat, err := os.ReadFile(path)
	check(err)
	return dat
}
