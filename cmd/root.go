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
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const Version = "1.0.0"

type LocalizationFile struct {
	file       string
	localeCode string
}
type CredentialsYaml struct {
	ApiToken string `yaml:"api_token"`
}

var fileTypesOpt = []string{
	"android_xml",
	"ios_strings",
	"ios_stringsdict",
	"java_properties",
	"rails_yaml",
	"angular_xlf",
	"flutter_arb",
	"dotnet_resx",
	"po",
	"pot",
	"json",
	"csv",
	"xlsx",
}

var javaEncodingOpt = []string{
	"utf_8",
	"latin_1",
}

var exportEmptyAsOpt = []string{
	"empty",
	"main",
	"skip",
}

func formatOptions(options []string, columns int) string {
	formatted := ""

	for k, v := range options {
		if k%columns == 0 {
			formatted += "\n"
		}
		formatted += "- "
		formatted += v
		formatted += strings.Repeat(" ", 20-len(v))
	}

	return formatted
}

var rootCmd = &cobra.Command{
	Use:     "localizely-cli",
	Short:   "Localizely is a translation management system (TMS) that helps teams to automate, manage and translate content.",
	Version: Version,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
	viper.SetConfigName("localizely")

	apiToken, err := getApiToken()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Failed to read api token from the '%s'\n%v\n", filepath.Join("~", ".localizely", "credentials.yaml"), err)
		}
	}

	viper.SetDefault("api_token", apiToken)
	viper.SetDefault("upload.files", []interface{}{})
	viper.SetDefault("download.files", []interface{}{})

	viper.SetEnvPrefix("LOCALIZELY")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func getApiToken() (string, error) {
	var err error

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, ".localizely", "credentials.yaml")

	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var credentialsYaml CredentialsYaml
	err = yaml.Unmarshal(b, &credentialsYaml)
	if err != nil {
		return "", err
	}

	return credentialsYaml.ApiToken, nil
}

func convertFilesConfigToLocalizationFiles(files []interface{}, localizationFiles *[]LocalizationFile) {
	for _, v := range files {
		*localizationFiles = append(*localizationFiles, LocalizationFile{
			file:       (v.(map[string]interface{})["file"]).(string),
			localeCode: (v.(map[string]interface{})["locale_code"]).(string),
		})
	}
}

func convertFilesFlagToLocalizationFiles(files map[string]interface{}, localizationFiles *[]LocalizationFile) {
	fileRegexp := regexp.MustCompile(`file\[(?P<fid>\d+)\]`)
	localeCodeRegexp := regexp.MustCompile(`locale_code\[(?P<lcid>\d+)\]`)

	params := map[string]map[string]string{}

	for k, v := range files {
		fileRegexpMatch := fileRegexp.FindStringSubmatch(k)
		if len(fileRegexpMatch) > 1 {
			fileId := fileRegexpMatch[1]

			if params[fileId] == nil {
				params[fileId] = map[string]string{}
			}

			params[fileId]["file"] = v.(string)
		}

		localeCodeRegexpMatch := localeCodeRegexp.FindStringSubmatch(k)
		if len(localeCodeRegexpMatch) > 1 {
			localeCodeId := localeCodeRegexpMatch[1]

			if params[localeCodeId] == nil {
				params[localeCodeId] = map[string]string{}
			}

			params[localeCodeId]["locale_code"] = v.(string)
		}
	}

	for _, v := range params {
		if len(v) == 2 {
			*localizationFiles = append(*localizationFiles, LocalizationFile{
				file:       v["file"],
				localeCode: v["locale_code"],
			})
		}
	}
}

func validateApiToken(apiToken string) error {
	if apiToken == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "~"
		}

		credentialsFilePath := filepath.Join(home, ".localizely", "credentials.yaml")
		msg := fmt.Sprintf("The API token was not provided.\n\nPlease set it using one of the available options:\n- %s file\n- LOCALIZELY_API_TOKEN environment variable\n- api-token flag\n\nTo create a new API token, please visit https://app.localizely.com/account.\n\n", credentialsFilePath)
		return errors.New(msg)
	}

	return nil
}

func validateProjectId(projectId string) error {
	if projectId == "" {
		msg := fmt.Sprintf("The project ID was not provided.\n\nPlease set it using one of the available options:\n- localizely.yml file\n- LOCALIZELY_PROJECT_ID environment variable\n- project-id flag\n\nTo find your project ID, please visit https://app.localizely.com/projects\n\n")
		return errors.New(msg)
	}

	return nil
}

func validateFileType(fileType string) error {
	if fileType == "" {
		msg := fmt.Sprintf("The file type was not provided.\n\nPlease set it using one of the available options:\n- localizely.yml file\n- LOCALIZELY_FILE_TYPE environment variable\n- file-type flag\n\nAvailable file types: %s\n\n", formatOptions(fileTypesOpt, 2))
		return errors.New(msg)
	}

	for _, ft := range fileTypesOpt {
		if ft == fileType {
			return nil
		}
	}

	msg := fmt.Sprintf("The file type has invalid value.\n\nAvailable file types: %s\n\n", formatOptions(fileTypesOpt, 2))
	return errors.New(msg)
}

func validateFiles(files []LocalizationFile, command string) error {
	if len(files) == 0 {
		msg := fmt.Sprintf("The list of localization files for %s was not provided.\n\nPlease set it using of of the available options:\n- localizely.yml\n- files flag\n\n", command)
		return errors.New(msg)
	}

	return nil
}

func checkError(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
