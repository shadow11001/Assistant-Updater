package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var appVersion = "dev" // Overridden at build time via ldflags

func main() {
	logToDisk("Starting Assistant Updater (Version: %s)...", appVersion)

	var errorMessages []string
	var successMessages []string

	// 1. Decrypt Master Token
	masterToken, err := decryptToken()
	if err != nil {
		logToDisk("Warning: Failed to decrypt master token: %v\n", err)
		masterToken = ""
	}

	// 2. Fetch Central Config
	logToDisk("Fetching remote configuration...")
	config, err := fetchRemoteConfig(masterToken)
	if err != nil {
		errMsg := fmt.Sprintf("Fatal: Cannot fetch configuration: %v\n", err)
		logToDisk(errMsg)
		showMessage("Updater Error", errMsg, true)
		os.Exit(1)
	}

	// 3. Self-Destruct Check
	if config.Delete {
		logToDisk("Self-destruct flag detected. Triggering deletion routine.")
		if err := selfDestruct(); err != nil {
			logToDisk("Failed to execute self-destruct: %v", err)
		}
		os.Exit(0) // Should not reach here if selfDestruct succeeds, but just in case
	}

		// 4. Self-Update Check
	logToDisk("Checking for self-updates...")
	selfRelease, err := getLatestRelease("shadow11001", "Assistant-Updater", masterToken)
	if err == nil {
		remoteVer := strings.TrimPrefix(selfRelease.TagName, "v")
		localVer := strings.TrimPrefix(appVersion, "v")

		if localVer != "dev" && localVer != remoteVer {
			logToDisk("Self-update found: v%s -> v%s. Updating...", localVer, remoteVer)

			// Find the updater.zip asset (since we now upload a zip to circumvent firewalls)
			var updaterDownloadUrl string
			for _, asset := range selfRelease.Assets {
				if strings.ToLower(asset.Name) == "updater.zip" {
					updaterDownloadUrl = asset.URL
					break
				}
			}

			// If the manual updater.zip asset is missing/fails, fallback to the zipball (which now contains it)
			if updaterDownloadUrl == "" && selfRelease.ZipballURL != "" {
				logToDisk("No attached updater.zip found, falling back to zipball.")
				updaterDownloadUrl = selfRelease.ZipballURL
			}

			if updaterDownloadUrl != "" {
				tempUpdaterZip := filepath.Join(os.TempDir(), "updater_update.zip")
				
				logToDisk("Downloading self-updater payload from: %s", updaterDownloadUrl)
				if err := downloadReleaseAsset(updaterDownloadUrl, masterToken, tempUpdaterZip); err == nil {
					// We successfully downloaded the zip.
					// Extract it to temp dir
					extractDir := filepath.Join(os.TempDir(), "updater_extracted")
					logToDisk("Extracting self-update zip to: %s", extractDir)
					
					if err := extractZip(tempUpdaterZip, extractDir); err == nil {
						newUpdaterPath := filepath.Join(extractDir, "updater.exe")
						logToDisk("Extracted successfully. Verifying expected exe exists at: %s", newUpdaterPath)
						
						if _, statErr := os.Stat(newUpdaterPath); statErr != nil {
							logToDisk("Could not find updater.exe inside extracted zip: %v", statErr)
						} else {
							// We must replace the running executable using the CMD hack
							exePath, _ := os.Executable()

							logToDisk("Hotswapping binaries. Replacing %s with %s", exePath, newUpdaterPath)
							// cmd sequence: wait 2s, move/overwrite old exe with new exe, then launch the new exe 
							cmdStr := fmt.Sprintf("ping 127.0.0.1 -n 3 > nul & move /Y \"%s\" \"%s\" & start \"\" \"%s\"", newUpdaterPath, exePath, exePath)
							logToDisk("Exec: %s", cmdStr)
							
							cmd := exec.Command("cmd.exe", "/C", cmdStr)

							if err := cmd.Start(); err == nil {
								logToDisk("Self-update downloaded and cmd spawned! Restarting app and terminating current process...")
								os.Exit(0)
							} else {
								logToDisk("Failed to spawn CMD update process: %v", err)
							}
						}
					} else {
						logToDisk("Failed to extract self-update asset: %v", err)
					}
				} else {
					logToDisk("Failed to download self-update asset zip: %v", err)
				}
			}
		} else {
			logToDisk("Warning: Failed to check for self-update: %v", err)
		}

		// 5. Update Loop for Apps
		docsDir, err := getDocumentsDir()
		if err != nil {
			errMsg := fmt.Sprintf("Failed to resolve Documents directory: %v", err)
			logToDisk(errMsg)
			showMessage("Updater Error", errMsg, true)
			os.Exit(1)
		}

		for _, app := range config.Apps {
			logToDisk("\n--- Processing %s ---", app.Name)

			// Determine the token to use for this app
			appToken := app.Token
			if appToken == "" {
				appToken = masterToken
			}

			// Check current local version
			currentVer, err := getCurrentVersion(docsDir, app)
			if err != nil {
				logToDisk("Warning: Could not parse current version (maybe it's missing or corrupted): %v", err)
				currentVer = "" // Treat as needing install
			}

			if currentVer != "" {
				logToDisk("Current local version: %s", currentVer)
			} else {
				logToDisk("App not found or no version info. Proceeding to install/update.")
			}

			// Fetch latest release info
			release, err := getLatestRelease(app.Owner, app.Repo, appToken)
			if err != nil {
				errMsg := fmt.Sprintf("Error: Failed to fetch latest release for %s/%s: %v", app.Owner, app.Repo, err)
				logToDisk(errMsg)
				errorMessages = append(errorMessages, errMsg)
				continue
			}

			logToDisk("Latest remote version: %s", release.TagName)

			// Normalize versions for basic string comparison (strip leading 'v' if present)
			localV := strings.TrimPrefix(currentVer, "v")
			remoteV := strings.TrimPrefix(release.TagName, "v")

			if localV == remoteV && localV != "" {
				logToDisk("App %s is up to date.", app.Name)
				continue
			}

			logToDisk("Update required for %s. Finding ZIP asset...", app.Name)

			// Find the ZIP asset
			var zipDownloadUrl string
			// Fallback to the auto-generated Source Code Zipball if no explicit .zip asset is attached to the release
			if release.ZipballURL != "" {
				zipDownloadUrl = release.ZipballURL
			}

			for _, asset := range release.Assets {
				if strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
					// GitHub standard API provides an asset URL we can stream from using Accept headers
					zipDownloadUrl = asset.URL
					break
				}
			}

			if zipDownloadUrl == "" {
				errMsg := fmt.Sprintf("Error: No .zip asset or source code zipball found in the latest release of %s.", app.Name)
				logToDisk(errMsg)
				errorMessages = append(errorMessages, errMsg)
				continue
			}

			// Prepare temp file path for the downloaded zip
			tempZipPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s_%s.zip", app.Name, release.TagName))

			// Download
			logToDisk("Downloading asset...")
			if err := downloadReleaseAsset(zipDownloadUrl, appToken, tempZipPath); err != nil {
				errMsg := fmt.Sprintf("Error: Failed to download asset: %v", err)
				logToDisk(errMsg)
				errorMessages = append(errorMessages, errMsg)
				continue
			}

			// Extract over the existing destination
			destPath := filepath.Join(docsDir, app.Name)
			logToDisk("Extracting to %s...", destPath)

			if err := extractZip(tempZipPath, destPath); err != nil {
				errMsg := fmt.Sprintf("Error: Failed to extract zip: %v", err)
				logToDisk(errMsg)
				errorMessages = append(errorMessages, errMsg)
				continue
			}

			// Cleanup downloaded zip
			os.Remove(tempZipPath)

			successMsg := fmt.Sprintf("Successfully updated %s to %s.", app.Name, release.TagName)
			logToDisk(successMsg)
			successMessages = append(successMessages, successMsg)
		}

		logToDisk("\nUpdate process complete.")

		// Construct final dialog message
		if len(errorMessages) > 0 {
			finalMsg := "Update finished with errors:\n\n" + strings.Join(errorMessages, "\n")
			if len(successMessages) > 0 {
				finalMsg += "\n\nSuccessful updates:\n" + strings.Join(successMessages, "\n")
			}
			showMessage("Assistant Updater - Issues Detected", finalMsg, true)
		} else if len(successMessages) > 0 {
			finalMsg := "Successfully applied updates:\n\n" + strings.Join(successMessages, "\n")
			showMessage("Assistant Updater", finalMsg, false)
		}
	}
}
