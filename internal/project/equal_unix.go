//go:build !windows

package project

func pathsEqual(left, right string) bool {
	return left == right
}
