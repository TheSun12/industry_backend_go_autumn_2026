package main
import (
	"strings"
)

func greet(name string) string {
	s := strings.TrimSpace(name);

	if s == "" {
		s = "World"
	}

	return "Hello, " + s + "!"
}
