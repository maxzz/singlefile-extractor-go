package main

import "strings"

func hasValidInputFlag(args []string) bool {
	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		if a == "" {
			continue
		}
		if a == "--" {
			break
		}
		if !strings.HasPrefix(a, "-") {
			continue
		}

		trimmed := strings.TrimLeft(a, "-")
		name, val, hasEq := strings.Cut(trimmed, "=")
		name = strings.ToLower(strings.TrimSpace(name))

		if name != "i" && name != "input" {
			continue
		}

		if hasEq {
			return strings.TrimSpace(val) != ""
		}

		// Value should be in the next arg.
		if i+1 >= len(args) {
			return false
		}
		next := strings.TrimSpace(args[i+1])
		if next == "" {
			return false
		}
		// Treat another flag as missing value (common mistake: --input --help).
		if strings.HasPrefix(next, "-") {
			return false
		}
		return true
	}
	return false
}

