package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

var launchReport = func(path string) error {
	command, args, err := reportOpenCommand(runtime.GOOS, path)
	if err != nil {
		return err
	}
	return exec.Command(command, args...).Start()
}

func reportOpenCommand(goos, path string) (string, []string, error) {
	switch goos {
	case "windows":
		return "explorer.exe", []string{path}, nil
	case "darwin":
		return "open", []string{path}, nil
	case "linux":
		return "xdg-open", []string{path}, nil
	default:
		return "", nil, fmt.Errorf("opening reports is unsupported on %s", goos)
	}
}
