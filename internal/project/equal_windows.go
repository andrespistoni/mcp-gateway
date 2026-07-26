//go:build windows

package project

import "strings"

func pathsEqual(left, right string) bool {
	return strings.EqualFold(left, right)
}
