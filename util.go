package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func DoesExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return false
	}
	return true
}

func findExecutablePath(exe string) (string, error) {
	pathFolders := strings.Split(os.Getenv("PATH"), ":")
	for _, val := range pathFolders {
		exepath := filepath.Join(val, exe)
		if DoesExist(exepath) {
			return exepath, nil
		}
	}
	return "", errors.New("executable not found in PATH")
}
