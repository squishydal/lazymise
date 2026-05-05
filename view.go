package main

import "fmt"

func (m model) View() string {
	s := fmt.Sprintf("Counter: %d\n\n", m.counter)
	s += "Your Tools:\n\n"

	for toolName, version := range m.tools {
		s += fmt.Sprintf("  -> %s: %s\n", toolName, version)
	}

	s += "\nPress up/down to change, q to quit."
	return s
}
