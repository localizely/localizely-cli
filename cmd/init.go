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
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

const BaseLocalizelyYamlTemplate = `
# For more configuration details, see https://localizely.com/configuration-file/
config_version: 1.0 # Required. Only 1.0 available
project_id: {{ .ProjectId }} # Required. Your project ID from: https://app.localizely.com/projects
file_type: {{ .FileType }} # Required. Available values : android_xml, ios_strings, ios_stringsdict, java_properties, rails_yaml, angular_xlf, flutter_arb, dotnet_resx, po, pot, json, csv, xlsx
upload: # Required.
  files: # Required. List of files for upload to Localizely. Usually, it is just one file used for the main locale{{ range .UploadFiles }}
    - file: {{ .File }} # Required. Path to the translation file
      locale_code: {{ .LocaleCode }} # Required. Locale code for the file. Examples: en, de-DE, zh-Hans-CN{{ end }}
download: # Required.
  files: # Required. List of files for download from Localizely.{{ range .DownloadFiles }}
    - file: {{ .File }} # Required. Path to the translation file
      locale_code: {{ .LocaleCode }} # Required. Locale code for the file. Examples: en, de-DE, zh-Hans-CN{{ end }}
`

const LocalizelyYamlTemplate = `
config_version: 1.0 # Required. Only 1.0 available
project_id: c776c33e-f428-4c91-87e1-a6a18c1554fe # Required. Your project ID from: https://app.localizely.com/projects
file_type: flutter_arb # Required. Available values : android_xml, ios_strings, ios_stringsdict, java_properties, rails_yaml, angular_xlf, flutter_arb, dotnet_resx, po, pot, json, csv, xlsx
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

func scanApiToken(apiToken *string) error {
	for {
		err := scan("\nEnter your API token (from https://app.localizely.com/account):", apiToken)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to read API token\nError: %v\n", err))
		}

		*apiToken = strings.TrimSpace(*apiToken)

		if len(*apiToken) > 0 {
			break
		}

		color.Set(color.FgRed)
		fmt.Fprintf(os.Stderr, "Invalid API token provided\n")
		color.Unset()
	}

	return nil
}

func scanProjectId(projectId *string) error {
	for {
		err := scan("\nEnter your project ID (from https://app.localizely.com/projects):", projectId)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to read project ID\nError: %v\n", err))
		}

		*projectId = strings.TrimSpace(*projectId)

		if len(*projectId) > 0 {
			break
		}

		color.Set(color.FgRed)
		fmt.Fprintf(os.Stderr, "Invalid project ID provided\n")
		color.Unset()
	}

	return nil
}

func scanFileType(fileType *string) error {
	for {
		var indexStr string
		var err error

		err = scan(fmt.Sprintf("\n%s\nSelect file type (%d-%d):", formatOptions(fileTypesOpt, 1, "ordered"), 1, len(fileTypesOpt)), &indexStr)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to read file type\nError: %v\n", err))
		}

		index, err := strconv.Atoi(strings.TrimSpace(indexStr))
		if err == nil && index >= 1 && index <= len(fileTypesOpt) {
			*fileType = fileTypesOpt[index-1]
			break
		}

		color.Set(color.FgRed)
		fmt.Fprintf(os.Stderr, "Invalid file type provided\n")
		color.Unset()
	}

	return nil
}

func scanFiles(localizationFiles *[]LocalizationFile, section string) error {
	localeCodeRegexp := regexp.MustCompile("^[a-z]{2,3}(-[A-Z][a-z]{3})?(-([A-Z]{2}|[0-9]{3}))?$")

	var action string
	if section == "pull" {
		action = "pull from Localizely"
	} else {
		action = "push to Localizely"
	}

	for {
		var localeCode string
		var file string
		var next string

		fmt.Println()

		for {
			err := scan(fmt.Sprintf("Enter locale code of the file you would like to %s (e.g. en, fr-FR, zh-Hans-CN):", action), &localeCode)
			if err != nil {
				return errors.New(fmt.Sprintf("Failed to read locale code\nError: %v\n", err))
			}

			localeCode = strings.TrimSpace(localeCode)
			if match := localeCodeRegexp.MatchString(localeCode); match == true {
				break
			}

			color.Set(color.FgRed)
			fmt.Fprintf(os.Stderr, "Invalid locale code provided\n")
			color.Unset()
		}

		for {
			err := scan(fmt.Sprintf("Enter the file path for the '%s' locale code (e.g. lang/en.json):", localeCode), &file)
			if err != nil {
				return errors.New(fmt.Sprintf("Failed to read file path\nError: %v\n", err))
			}

			file = strings.TrimSpace(file)

			if len(file) >= 1 {
				break
			}

			color.Set(color.FgRed)
			fmt.Fprintf(os.Stderr, "Invalid file path provided\n")
			color.Unset()
		}

		*localizationFiles = append(*localizationFiles, LocalizationFile{File: file, LocaleCode: localeCode})

		for {
			err := scan(fmt.Sprintf("Add another localization file for %s? (y/n)", section), &next)
			if err != nil {
				return errors.New(fmt.Sprintf("Failed to read answer\nError: %v\n", err))
			}

			next = strings.ToLower(strings.TrimSpace(next))

			if next == "n" || next == "y" {
				break
			}

			color.Set(color.FgRed)
			fmt.Fprintf(os.Stderr, "Invalid answer provided\n")
			color.Unset()
		}

		if next == "n" {
			break
		}
	}

	return nil
}

func scan(message string, value *string) error {
	fmt.Print(message + " ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	*value = line

	return nil
}

func createCredentialsYamlFile(apiToken string) error {
	bytes, err := yaml.Marshal(CredentialsYaml{ApiToken: apiToken})
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := filepath.Join(home, LocalizelyDir, CredentialsYamlFile)

	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "\nOverwriting the content of the '%s' file\n", CredentialsYamlFile)
	}

	return os.WriteFile(path, bytes, 0666)
}

func createLocalizelyYamlFile(projectId string, fileType string, uploadFiles []LocalizationFile, downloadFiles []LocalizationFile) error {
	tmpl, err := template.New("localizelyYaml").Parse(strings.TrimSpace(BaseLocalizelyYamlTemplate))
	if err != nil {
		return err
	}

	file, err := os.Create(LocalizelyYamlFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, BaseLocalizelyYaml{
		ProjectId:     projectId,
		FileType:      fileType,
		UploadFiles:   uploadFiles,
		DownloadFiles: downloadFiles,
	})
}

func checkIsConfigured() error {
	if _, err := os.Stat(LocalizelyYamlFile); !errors.Is(err, os.ErrNotExist) {
		return errors.New(fmt.Sprintf("Localizely client is already configured\nTo see configuration, please open the '%s' file\nFor more configuration details, see https://localizely.com/configuration-file/\n", LocalizelyYamlFile))
	}

	return nil
}

func initInteractive() error {
	fmt.Printf("\nRunning init command in interactive mode\n")
	var err error

	var apiToken string
	err = scanApiToken(&apiToken)
	if err != nil {
		return err
	}

	var projectId string
	err = scanProjectId(&projectId)
	if err != nil {
		return err
	}

	var fileType string
	err = scanFileType(&fileType)
	if err != nil {
		return err
	}

	var uploadFiles []LocalizationFile
	err = scanFiles(&uploadFiles, "push")
	if err != nil {
		return err
	}

	var downloadFiles []LocalizationFile
	err = scanFiles(&downloadFiles, "pull")
	if err != nil {
		return err
	}

	err = createCredentialsYamlFile(apiToken)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to save api token\nError: %v\n", err))
	}

	color.Green("\nSuccessfully saved api token in the '%s' file", formatCredentialsYamlFilePath())

	err = createLocalizelyYamlFile(projectId, fileType, uploadFiles, downloadFiles)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to create '%s'\nError: %v\n", LocalizelyYamlFile, err))
	}

	color.Green("\nSuccessfully created '%s' file\nFor advanced configuration options, see https://localizely.com/configuration-file/", LocalizelyYamlFile)

	return nil
}

func initTemplate() error {
	data := []byte(strings.TrimSpace(LocalizelyYamlTemplate))

	err := os.WriteFile(LocalizelyYamlFile, data, 0666)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to generate template file\nError: %v\n", err))
	}

	color.Green("\nSuccessfully generated the '%s' template file\nFor more configuration details, see https://localizely.com/configuration-file/", LocalizelyYamlFile)

	return nil
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure your Localizely client",
	Long:  "Configure your Localizely client\n(Learn more here https://localizely.com/configuration-file/)\n",
	Run: func(cmd *cobra.Command, args []string) {
		mode, err := cmd.Flags().GetString("mode")
		checkError(err)

		err = validateMode(mode)
		checkError(err)

		err = checkIsConfigured()
		checkError(err)

		fmt.Print(LocalizelyLogo)

		if mode == "template" {
			err = initTemplate()
			checkError(err)
		} else {
			err = initInteractive()
			checkError(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().String("mode", "", "Configuration mode (default \"interactive\")\n"+formatOptions(modeOpt, 1, "unordered"))
}
