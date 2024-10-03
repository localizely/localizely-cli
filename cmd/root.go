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

const Version = "1.0.6"

const LocalizelyDir = ".localizely"

const LocalizelyYamlFile = "localizely.yml"

const CredentialsYamlFile = "credentials.yaml"

type LocalizationFile struct {
	File       string
	LocaleCode string
}
type CredentialsYaml struct {
	ApiToken string `yaml:"api_token"`
}

type BaseLocalizelyYaml struct {
	ProjectId     string
	FileType      string
	UploadFiles   []LocalizationFile
	DownloadFiles []LocalizationFile
}

var modeOpt = []string{
	"interactive",
	"template",
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

var javaPropertiesEncodingOpt = []string{
	"utf_8",
	"latin_1",
}

var exportEmptyAsOpt = []string{
	"empty",
	"main",
	"skip",
}

var rootCmd = &cobra.Command{
	Use:     "localizely-cli",
	Short:   "Localizely is a translation management platform that helps you translate texts in your app for targeting multilingual market.",
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
			fmt.Fprintf(os.Stderr, "Failed to read api token from the '%s'\nError: %v\n", formatCredentialsYamlFilePath(), err)
		}
	}

	viper.SetDefault("api_token", apiToken)
	viper.SetDefault("upload.files", []interface{}{})
	viper.SetDefault("download.files", []interface{}{})

	viper.SetEnvPrefix("LOCALIZELY")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintf(os.Stderr, "Using config file: '%s'\n", viper.ConfigFileUsed())
	}
}

func formatOptions(options []string, columns int, mode string) string {
	formatted := ""

	for k, v := range options {
		if k != 0 && k%columns == 0 {
			formatted += "\n"
		}

		if mode == "ordered" {
			formatted += fmt.Sprintf("%d. ", k+1)
		} else {
			formatted += "- "
		}
		formatted += v
		formatted += strings.Repeat(" ", 20-len(v))
	}

	return formatted
}

func formatCredentialsYamlFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, LocalizelyDir, CredentialsYamlFile)
}

func getApiToken() (string, error) {
	var err error

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(home, LocalizelyDir, CredentialsYamlFile)

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
			File:       (v.(map[string]interface{})["file"]).(string),
			LocaleCode: (v.(map[string]interface{})["locale_code"]).(string),
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
				File:       v["file"],
				LocaleCode: v["locale_code"],
			})
		}
	}
}

func validateApiToken(apiToken string) error {
	if apiToken == "" {
		msg := fmt.Sprintf("The API token was not provided.\n\nPlease set it using one of the available options:\n- %s file\n- LOCALIZELY_API_TOKEN environment variable\n- api-token flag\n\nTo create a new API token, please visit https://app.localizely.com/account.\n\nUse \"localizely-cli [command] --help\" for more information about a command.\n\n", formatCredentialsYamlFilePath())
		return errors.New(msg)
	}

	return nil
}

func validateProjectId(projectId string) error {
	if projectId == "" {
		msg := fmt.Sprintf("The project ID was not provided.\n\nPlease set it using one of the available options:\n- %s file (Learn more here https://localizely.com/configuration-file/)\n- LOCALIZELY_PROJECT_ID environment variable\n- project-id flag\n\nTo find your project ID, please visit https://app.localizely.com/projects\n\nUse \"localizely-cli [command] --help\" for more information about a command.\n\n", LocalizelyYamlFile)
		return errors.New(msg)
	}

	return nil
}

func validateFileType(fileType string) error {
	if fileType == "" {
		msg := fmt.Sprintf("The file type was not provided.\n\nPlease set it using one of the available options:\n- %s file (Learn more here https://localizely.com/configuration-file/)\n- LOCALIZELY_FILE_TYPE environment variable\n- file-type flag\n\nAvailable file types:\n%s\n\nUse \"localizely-cli [command] --help\" for more information about a command.\n\n", LocalizelyYamlFile, formatOptions(fileTypesOpt, 2, "unordered"))
		return errors.New(msg)
	}

	for _, ft := range fileTypesOpt {
		if ft == fileType {
			return nil
		}
	}

	msg := fmt.Sprintf("The file type has invalid value.\n\nAvailable file types:\n%s\n\nUse \"localizely-cli [command] --help\" for more information about a command.\n\n", formatOptions(fileTypesOpt, 2, "unordered"))
	return errors.New(msg)
}

func validateFiles(files []LocalizationFile, command string) error {
	if len(files) == 0 {
		msg := fmt.Sprintf("The list of localization files for %s was not provided.\n\nPlease set it using one of the available options:\n- %s file (Learn more here https://localizely.com/configuration-file/)\n- files flag\n\nUse \"localizely-cli [command] --help\" for more information about a command.\n\n", command, LocalizelyYamlFile)
		return errors.New(msg)
	}

	return nil
}

func validateExportEmptyAs(exportEmptyAs string) error {
	if exportEmptyAs == "" {
		return nil
	}

	for _, opt := range exportEmptyAsOpt {
		if opt == exportEmptyAs {
			return nil
		}
	}

	msg := fmt.Sprintf("The export-empty-as has invalid value.\n\nAvailable options:\n%s\n\nUse \"localizely-cli [command] --help\" for more information about a command.\n\n", formatOptions(exportEmptyAsOpt, 1, "unordered"))
	return errors.New(msg)
}

func validateJavaPropertiesEncoding(javaPropertiesEncoding string) error {
	if javaPropertiesEncoding == "" {
		return nil
	}

	for _, opt := range javaPropertiesEncodingOpt {
		if opt == javaPropertiesEncoding {
			return nil
		}
	}

	msg := fmt.Sprintf("The java properties encoding has invalid value.\n\nAvailable options:\n%s\n\nUse \"localizely-cli [command] --help\" for more information about a command.\n\n", formatOptions(javaPropertiesEncodingOpt, 1, "unordered"))
	return errors.New(msg)
}

func validateMode(mode string) error {
	if mode == "" {
		return nil
	}

	for _, opt := range modeOpt {
		if opt == mode {
			return nil
		}
	}

	msg := fmt.Sprintf("The mode has invalid value.\n\nAvailable mode options:\n%s\n\nUse \"localizely-cli [command] --help\" for more information about a command.\n\n", formatOptions(modeOpt, 1, "unordered"))
	return errors.New(msg)
}

func checkError(err error) {
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}
