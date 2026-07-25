package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// getCurrentVersion reads the designated config file and attempts to extract a version string
// For 'js/defaultConfig.js', we assume something like: version: "v1.0.0"
func getCurrentVersion(docsDir string, app AppTarget) (string, error) {
	configPath := filepath.Join(docsDir, app.Name, filepath.FromSlash(app.VersionCheckFile))

	bytes, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // App doesn't exist, we will install it
		}
		return "", err
	}

	// This regex looks for 'version' followed by quotes. E.g., version: "v1.2.3" or version="1.2.3"
	// It extracts the contents of the quotes.
	re := regexp.MustCompile(`version\s*[:=]\s*["']([^"']+)["']`)
	matches := re.FindSubmatch(bytes)

	if len(matches) > 1 {
		return string(matches[1]), nil
	}

	return "", fmt.Errorf("no version string found in %s", configPath)
}

// selfDestruct spawns a detached cmd.exe process that waits 2 seconds,
// deletes the current executable, and then exits. The Go process terminates immediately.
func selfDestruct() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// Windows trick to delete a running executable:
	// We spawn a cmd that pings localhost to delay, deletes the file, and exits.
	cmdStr := fmt.Sprintf("ping 127.0.0.1 -n 3 > nul & del /F /Q \"%s\"", exePath)
	cmd := exec.Command("cmd.exe", "/C", cmdStr)

	// Start detaches the process, allowing the Go program to exit cleanly.
	if err := cmd.Start(); err != nil {
		return err
	}

	// Exit our application before the ping finishes
	os.Exit(0)
	return nil
}
