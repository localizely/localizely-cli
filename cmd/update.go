/*
Copyright © 2022 Localizely

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/fatih/color"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

func scanConfirmUpdate(confirm *bool) {
	var confirmStr string

	for {
		err := scan("Do you want to proceed with the update? (y/n)", &confirmStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read answer\nError: %v\n", err)
			os.Exit(1)
		}

		confirmStr = strings.ToLower(strings.TrimSpace(confirmStr))

		if confirmStr == "y" || confirmStr == "n" {
			break
		}

		color.Set(color.FgRed)
		fmt.Fprintf(os.Stderr, "Invalid answer provided\n")
		color.Unset()
	}

	*confirm = confirmStr == "y"
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Localizely CLI to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		currVersion := semver.MustParse(Version)

		latest, found, err := selfupdate.DetectLatest("localizely/localizely-cli")
		if err != nil || !found {
			fmt.Fprintf(os.Stderr, "Failed to detect the latest release\nError: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Current version: %s\n", currVersion)
		fmt.Printf("Latest version:  %s\n\n", latest.Version)

		if currVersion.GE(latest.Version) {
			color.Green("Localizely CLI is up to date")
		} else {
			var confirm bool
			scanConfirmUpdate(&confirm)

			if !confirm {
				fmt.Println("Update canceled")
				return
			}

			exe, err := os.Executable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to locate executable path\nError: %v\n", err)
				os.Exit(1)
			}

			err = selfupdate.UpdateTo(latest.AssetURL, exe)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to update\nError: %v\n", err)
				os.Exit(1)
			}

			color.Green("Successfully updated to %s", latest.Version)
		}
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
