package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("Starting Assistant Updater...")

	// 1. Decrypt Master Token
	masterToken, err := decryptToken()
	if err != nil {
		log.Printf("Warning: Failed to decrypt master token: %v\n", err)
		log.Println("Will attempt to fetch config unauthenticated if possible.")
		masterToken = ""
	}

	// 2. Fetch Central Config
	fmt.Println("Fetching remote configuration...")
	config, err := fetchRemoteConfig(masterToken)
	if err != nil {
		log.Fatalf("Fatal: Cannot fetch configuration: %v\n", err)
	}

	// 3. Self-Destruct Check
	if config.Delete {
		fmt.Println("Self-destruct flag detected. Triggering deletion routine.")
		if err := selfDestruct(); err != nil {
			log.Fatalf("Failed to execute self-destruct: %v", err)
		}
		os.Exit(0) // Should not reach here if selfDestruct succeeds, but just in case
	}

	// 4. Update Loop for Apps
	docsDir, err := getDocumentsDir()
	if err != nil {
		log.Fatalf("Failed to resolve Documents directory: %v", err)
	}

	for _, app := range config.Apps {
		fmt.Printf("\n--- Processing %s ---\n", app.Name)

		// Determine the token to use for this app
		appToken := app.Token
		if appToken == "" {
			appToken = masterToken
		}

		// Check current local version
		currentVer, err := getCurrentVersion(docsDir, app)
		if err != nil {
			fmt.Printf("Warning: Could not parse current version (maybe it's missing or corrupted): %v\n", err)
			currentVer = "" // Treat as needing install
		}

		if currentVer != "" {
			fmt.Printf("Current local version: %s\n", currentVer)
		} else {
			fmt.Println("App not found or no version info. Proceeding to install/update.")
		}

		// Fetch latest release info
		release, err := getLatestRelease(app.Owner, app.Repo, appToken)
		if err != nil {
			fmt.Printf("Error: Failed to fetch latest release for %s/%s: %v\n", app.Owner, app.Repo, err)
			continue
		}

		fmt.Printf("Latest remote version: %s\n", release.TagName)

		// Normalize versions for basic string comparison (strip leading 'v' if present)
		localV := strings.TrimPrefix(currentVer, "v")
		remoteV := strings.TrimPrefix(release.TagName, "v")

		if localV == remoteV && localV != "" {
			fmt.Printf("App %s is up to date.\n", app.Name)
			continue
		}

		fmt.Printf("Update required for %s. Finding ZIP asset...\n", app.Name)

		// Find the ZIP asset
		var zipDownloadUrl string
		for _, asset := range release.Assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
				// GitHub standard API provides an asset URL we can stream from using Accept headers
				zipDownloadUrl = asset.URL
				break
			}
		}

		if zipDownloadUrl == "" {
			fmt.Printf("Error: No .zip asset found in the latest release of %s.\n", app.Name)
			continue
		}

		// Prepare temp file path for the downloaded zip
		tempZipPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s.zip", app.Name, release.TagName))

		// Download
		fmt.Println("Downloading asset...")
		if err := downloadReleaseAsset(zipDownloadUrl, appToken, tempZipPath); err != nil {
			fmt.Printf("Error: Failed to download asset: %v\n", err)
			continue
		}

		// Extract over the existing destination
		destPath := filepath.Join(docsDir, app.Name)
		fmt.Printf("Extracting to %s...\n", destPath)

		if err := extractZip(tempZipPath, destPath); err != nil {
			fmt.Printf("Error: Failed to extract zip: %v\n", err)
			continue
		}

		// Cleanup downloaded zip
		os.Remove(tempZipPath)

		fmt.Printf("Successfully updated %s to %s.\n", app.Name, release.TagName)
	}

	fmt.Println("\nUpdate process complete.")
}
