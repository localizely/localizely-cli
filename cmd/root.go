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
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const Version = "1.0.10"

const LocalizelyDir = ".localizely"

const LocalizelyYamlFile = "localizely.yml"

const CredentialsYamlFile = "credentials.yaml"

// LocalizationFile is one item of upload.files or download.files. Besides the file and its locale, an item may
// override the top-level file type and the section parameters that route string keys by tags, so that one
// repository can hold the files of several platforms (see https://localizely.com/configuration-file/).
type LocalizationFile struct {
	File       string
	LocaleCode string
	// Overrides the top-level file_type for this file
	FileType string
	// Upload: override upload.params for this file. nil means not set.
	TagAdded   []string
	TagUpdated []string
	TagRemoved []string
	TagInFile  []string
	// Download: override download.params for this file. nil means not set.
	IncludeTags []string
	ExcludeTags []string
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
	"ios_xcstrings",
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

// File types that keep every locale in one file; their items carry no locale_code
var multiLocaleFileTypes = map[string]bool{
	"ios_xcstrings": true,
}

var placeholderFormatsOpt = []string{
	"printf_java",
	"printf_ios",
	"printf_c",
	"icu",
	"dotnet",
	"ruby",
	"i18next",
	"raw",
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
		item, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		*localizationFiles = append(*localizationFiles, LocalizationFile{
			File:        stringValue(item["file"]),
			LocaleCode:  stringValue(item["locale_code"]),
			FileType:    stringValue(item["file_type"]),
			TagAdded:    stringList(item["tag_added"]),
			TagUpdated:  stringList(item["tag_updated"]),
			TagRemoved:  stringList(item["tag_removed"]),
			TagInFile:   stringList(item["tag_in_file"]),
			IncludeTags: stringList(item["include_tags"]),
			ExcludeTags: stringList(item["exclude_tags"]),
		})
	}
}

func stringValue(value interface{}) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}

	return ""
}

// stringList reads a YAML list; nil when the key is absent, so that an item can tell "not set" from "none"
func stringList(value interface{}) []string {
	switch list := value.(type) {
	case []interface{}:
		result := []string{}
		for _, v := range list {
			if s := stringValue(v); s != "" {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return list
	case string:
		return []string{list}
	default:
		return nil
	}
}

// convertFilesFlagToLocalizationFiles reads the --files flag: file[i], locale_code[i] and file_type[i] parameters
func convertFilesFlagToLocalizationFiles(files map[string]interface{}, localizationFiles *[]LocalizationFile) {
	paramRegexp := regexp.MustCompile(`^(file|locale_code|file_type)\[(\d+)\]$`)

	params := map[int]map[string]string{}

	for k, v := range files {
		match := paramRegexp.FindStringSubmatch(strings.TrimSpace(k))
		if len(match) != 3 {
			continue
		}
		index, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}
		if params[index] == nil {
			params[index] = map[string]string{}
		}
		params[index][match[1]] = stringValue(v)
	}

	indexes := make([]int, 0, len(params))
	for index := range params {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	for _, index := range indexes {
		v := params[index]
		if v["file"] == "" {
			continue
		}
		*localizationFiles = append(*localizationFiles, LocalizationFile{
			File:       v["file"],
			LocaleCode: v["locale_code"],
			FileType:   v["file_type"],
		})
	}
}

// resolveFileType is the file type of an item: its own when set, otherwise the top-level one
func resolveFileType(file LocalizationFile, fileType string) string {
	if file.FileType != "" {
		return file.FileType
	}

	return fileType
}

// isMultiLocaleFile tells whether a file holds every locale, from its file type or, when none is configured for an
// upload, from its extension
func isMultiLocaleFile(file LocalizationFile, fileType string) bool {
	resolved := resolveFileType(file, fileType)
	if resolved != "" {
		return multiLocaleFileTypes[resolved]
	}

	return strings.EqualFold(filepath.Ext(file.File), ".xcstrings")
}

// resolveList is a file-level list when the item sets one, otherwise the section-level one
func resolveList(fileLevel []string, sectionLevel []string) []string {
	if fileLevel != nil {
		return fileLevel
	}

	return sectionLevel
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

func isValidFileType(fileType string) bool {
	for _, ft := range fileTypesOpt {
		if ft == fileType {
			return true
		}
	}

	return false
}

// validateFileType checks the top-level file type, which may be empty when every item sets its own
func validateFileType(fileType string) error {
	if fileType == "" || isValidFileType(fileType) {
		return nil
	}

	msg := fmt.Sprintf("The file type has invalid value.\n\nAvailable file types:\n%s\n\n", formatOptions(fileTypesOpt, 2, "unordered"))
	return errors.New(msg)
}

// validateFiles checks the items of a section against the top-level file type. A pull needs a file type for every
// file; a push reads the format from the file, so a file type is only needed to tell multi-locale files apart.
func validateFiles(files []LocalizationFile, fileType string, command string) error {
	if len(files) == 0 {
		msg := fmt.Sprintf("The list of localization files for %s was not provided.\n\nPlease set it using one of the available options:\n- %s file\n- files flag\n\n", command, LocalizelyYamlFile)
		return errors.New(msg)
	}

	seen := map[string]bool{}
	for _, file := range files {
		if file.File == "" {
			return errors.New(fmt.Sprintf("Every item of the list of localization files for %s must contain the 'file' parameter.\n\n", command))
		}
		if seen[file.File] {
			return errors.New(fmt.Sprintf("The file '%s' is listed more than once for %s.\n\n", file.File, command))
		}
		seen[file.File] = true

		resolved := resolveFileType(file, fileType)
		if resolved == "" && command == "pull" {
			msg := fmt.Sprintf("The file type was not provided for the file '%s'.\n\nPlease set it using one of the available options:\n- %s file, as the top-level file_type or as file_type of the item\n- LOCALIZELY_FILE_TYPE environment variable\n- file-type flag\n\nAvailable file types:\n%s\n\n", file.File, LocalizelyYamlFile, formatOptions(fileTypesOpt, 2, "unordered"))
			return errors.New(msg)
		}
		if resolved != "" && !isValidFileType(resolved) {
			msg := fmt.Sprintf("The file type '%s' of the file '%s' is invalid.\n\nAvailable file types:\n%s\n\n", resolved, file.File, formatOptions(fileTypesOpt, 2, "unordered"))
			return errors.New(msg)
		}

		if isMultiLocaleFile(file, fileType) {
			if file.LocaleCode != "" {
				return errors.New(fmt.Sprintf("The file '%s' contains all locales, so its 'locale_code' parameter must be omitted.\n\n", file.File))
			}
		} else if file.LocaleCode == "" {
			return errors.New(fmt.Sprintf("The 'locale_code' parameter is missing for the file '%s'.\n\n", file.File))
		}
	}

	return nil
}

func validatePlaceholderFormat(placeholderFormat string) error {
	if placeholderFormat == "" {
		return nil
	}

	for _, opt := range placeholderFormatsOpt {
		if opt == placeholderFormat {
			return nil
		}
	}

	msg := fmt.Sprintf("The placeholder-format has invalid value.\n\nAvailable options:\n%s\n\n", formatOptions(placeholderFormatsOpt, 1, "unordered"))
	return errors.New(msg)
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
