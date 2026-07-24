package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// AppTarget represents an app we want to keep up to date
type AppTarget struct {
	Name             string `json:"name"`              // Folder name in Documents, e.g., "Dispatch-Assistant"
	Owner            string `json:"owner"`             // GitHub Owner, e.g. "shadow11001"
	Repo             string `json:"repo"`              // GitHub Repo, e.g., "Dispatch-Assistant"
	VersionCheckFile string `json:"versionCheckFile"`  // Relative path to version file, e.g. "js/defaultConfig.js"
	Token            string `json:"token,omitempty"`   // Optional per-app token. If omitted, master token can be used if appropriate.
}

// RemoteConfig is the central JSON configuration pulled from the repository
type RemoteConfig struct {
	Delete bool        `json:"delete"`
	Apps   []AppTarget `json:"apps"`
}

// ConfigURL is the URL where the remote config.json is hosted.
// E.g. https://raw.githubusercontent.com/shadow11001/Assistant-Updater/main/config.json
const ConfigURL = "https://raw.githubusercontent.com/shadow11001/Assistant-Updater/main/config.json"

// getDocumentsDir dynamically finds the current user's Documents folder.
func getDocumentsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Note: In typical Windows setups, Documents is nested under the user's home directory.
	// For OneDrive setups or specialized network folders, this might need fallback logic or Windows API calls.
	docsPath := filepath.Join(home, "Documents")
	return docsPath, nil
}

// fetchRemoteConfig retrieves the central JSON configuration securely
func fetchRemoteConfig(token string) (*RemoteConfig, error) {
	req, err := http.NewRequest("GET", ConfigURL, nil)
	if err != nil {
		return nil, err
	}

	// Always send the Authorization header if we have a token
	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config endpoint returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var config RemoteConfig
	if err := json.Unmarshal(bodyBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse remote config: %v", err)
	}

	return &config, nil
}
