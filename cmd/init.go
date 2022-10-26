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
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const LocalizelyLogo = `
 _                       _ _            _       
| |                     | (_)          | |      
| |      ___   ____ ____| |_ _____ ____| |_   _ 
| |     / _ \ / ___) _  | | (___  ) _  ) | | | |
| |____| |_| ( (__( ( | | | |/ __( (/ /| | |_| |
|_______)___/ \____)_||_|_|_(_____)____)_|\__  |
                                         (____/ 
`

const LocalizelyYamlTemplate = `
config_version: 1.0 # Required. Only 1.0 available
project_id: c776c33e-f428-4c91-87e1-a6a18c1554fe # Required. Your project ID from: https://app.localizely.com/projects
file_type: flutter_arb # Required. Available values : android_xml, ios_strings, ios_stringsdict, java_properties, rails_yaml, angular_xlf, flutter_arb, dotnet_resx, po, json, csv
branch: main # Optional. Your branch in Localizely project to sync files with.
upload: # Required.
  files: # Required. List of files for upload to Localizely. Usually, it is just one file used for the main locale
    - file: lib/l10n/intl_en.arb # Required. Path to the translation file
      locale_code: en # Required. Locale code for the file. Examples: en, de-DE, zh-Hans-CN
  params: # Optional.
    overwrite: true # Optional, default: false. If the translation in a given language should be overwritten with modified translation from uploading file.
    reviewed: false # Optional, default: false. If uploading translations, that are added, should be marked as Reviewed. For uploading translations that are only modified it will have effect only if overwrite is set to true.
    tag_added: # Optional. List of tags to add to new translations from uploading file.
      - added
    tag_removed: # Optional. List of tags to add to removed translations from uploading file.
      - removed
    tag_updated: # Optional. List of tags to add to updated translations from uploading file.
      - updated
download: # Required.
  files: # Required. List of files for download from Localizely.
    - file: lib/l10n/intl_en.arb # Required. Path to the translation file
      locale_code: en # Required. Locale code for the file. Examples: en, de-DE, zh-Hans-CN
    - file: lib/l10n/intl_de.arb # Required. Path to the translation file
      locale_code: de # Required. Locale code for the file. Examples: en, de-DE, zh-Hans-CN
  params:
    export_empty_as: empty # Optional, default: empty. How you would like empty translations to be exported. Allowed values are 'empty' to keep empty, 'main' to replace with the main language value, or 'skip' to omit.
    exclude_tags: # Optional. List of tags to be excluded from the download. If not set, all string keys will be considered for download.
      - removed
    include_tags: # Optional. List of tags to be downloaded. If not set, all string keys will be considered for download.
      - new
    java_properties_encoding: utf_8 # Optional, default: latin_1. (Only for Java .properties files download) Character encoding. Available values : 'utf_8', 'latin_1'
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure your Localizely client",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat("localizely.yml"); !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "Localizely client is already configured\nTo see configuration, please open 'localizely.yml' file (more details - https://localizely.com/configuration-file/)")
			os.Exit(1)
		}

		fmt.Print(LocalizelyLogo)

		data := []byte(LocalizelyYamlTemplate)

		err := os.WriteFile("localizely.yml", data, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate template file\nError: %v", err)
			os.Exit(1)
		}

		fmt.Println("Successfully generated 'localizely.yml' template file (more details - https://localizely.com/configuration-file/)")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
