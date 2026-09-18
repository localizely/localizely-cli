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
	"net/url"
	"os"
	"path/filepath"
	"reflect"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull localization files from Localizely",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind flags only if the command is executed (fixes issue with global viper and the same flag names in multiple cobra commands)
		// More info: https://github.com/spf13/viper/issues/233#issuecomment-386791444
		viper.BindPFlag("api_token", cmd.Flags().Lookup("api-token"))
		viper.BindPFlag("project_id", cmd.Flags().Lookup("project-id"))
		viper.BindPFlag("branch", cmd.Flags().Lookup("branch"))
		viper.BindPFlag("file_type", cmd.Flags().Lookup("file-type"))
		viper.BindPFlag("download.files", cmd.Flags().Lookup("files"))
		viper.BindPFlag("download.params.java_properties_encoding", cmd.Flags().Lookup("java-properties-encoding"))
		viper.BindPFlag("download.params.export_empty_as", cmd.Flags().Lookup("export-empty-as"))
		viper.BindPFlag("download.params.include_tags", cmd.Flags().Lookup("include-tags"))
		viper.BindPFlag("download.params.exclude_tags", cmd.Flags().Lookup("exclude-tags"))
		viper.BindPFlag("download.params.placeholder_format", cmd.Flags().Lookup("placeholder-format"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		apiToken := viper.GetString("api_token")
		projectId := viper.GetString("project_id")
		branch := viper.GetString("branch")
		fileType := viper.GetString("file_type")
		javaPropertiesEncoding := viper.GetString("download.params.java_properties_encoding")
		files := viper.Get("download.files")
		exportEmptyAs := viper.GetString("download.params.export_empty_as")
		includeTags := viper.GetStringSlice("download.params.include_tags")
		excludeTags := viper.GetStringSlice("download.params.exclude_tags")
		placeholderFormat := viper.GetString("download.params.placeholder_format")

		localizationFiles := []LocalizationFile{}
		if reflect.TypeOf(files).String() == "[]interface {}" {
			convertFilesConfigToLocalizationFiles(files.([]interface{}), &localizationFiles)
		} else if reflect.TypeOf(files).String() == "map[string]interface {}" {
			convertFilesFlagToLocalizationFiles(files.(map[string]interface{}), &localizationFiles)
		}

		err := validateApiToken(apiToken)
		checkError(err)

		err = validateProjectId(projectId)
		checkError(err)

		err = validateFileType(fileType)
		checkError(err)

		err = validateFiles(localizationFiles, fileType, "pull")
		checkError(err)

		err = validateExportEmptyAs(exportEmptyAs)
		checkError(err)

		err = validateJavaPropertiesEncoding(javaPropertiesEncoding)
		checkError(err)

		err = validatePlaceholderFormat(placeholderFormat)
		checkError(err)

		api := newApiClient(apiToken)

		for _, localizationFile := range localizationFiles {
			params := url.Values{}
			params.Set("type", resolveFileType(localizationFile, fileType))
			// A file with all locales, such as an Apple String Catalog, is pulled with every locale of the project
			if !isMultiLocaleFile(localizationFile, fileType) {
				params.Set("lang_codes", localizationFile.LocaleCode)
			}
			if branch != "" {
				params.Set("branch", branch)
			}
			// File-level tags override the section parameters for this file
			addTags(params, "include_tags", resolveList(localizationFile.IncludeTags, includeTags))
			addTags(params, "exclude_tags", resolveList(localizationFile.ExcludeTags, excludeTags))
			if exportEmptyAs != "" {
				params.Set("export_empty_as", exportEmptyAs)
			}
			if javaPropertiesEncoding != "" {
				params.Set("java_properties_encoding", javaPropertiesEncoding)
			}
			if placeholderFormat != "" {
				params.Set("placeholder_format", placeholderFormat)
			}

			content, err := api.downloadFile(projectId, params)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to pull data from Localizely for '%s'\nError: %v\n", localizationFile.File, err)
				os.Exit(1)
			}

			err = os.MkdirAll(filepath.Dir(localizationFile.File), 0666)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create directory '%s'\nError: %v\n", filepath.Dir(localizationFile.File), err)
				os.Exit(1)
			}

			err = os.WriteFile(filepath.Clean(localizationFile.File), content, 0666)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to save localization file '%s'\nError: %v\n", filepath.Clean(localizationFile.File), err)
				os.Exit(1)
			}
		}

		color.Green("Successfully pulled data from Localizely")
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)

	pullCmd.Flags().String("api-token", "", "API token\nYour API token from https://app.localizely.com/account")
	pullCmd.Flags().String("project-id", "", "Project ID\nYour project ID from https://app.localizely.com/projects")
	pullCmd.Flags().String("branch", "", "Branch name\nBranch in Localizely project to sync files with")
	pullCmd.Flags().StringToString("files", map[string]string{}, "List of localization files to pull from Localizely\nExample:\n\t--files \"file[0]=lang/en_US.json\",\"locale_code[0]=en-US\"\nA file may set its own file type, and a file that holds all locales takes no locale_code[i]:\n\t--files \"file[0]=App/Localizable.xcstrings\",\"file_type[0]=ios_xcstrings\"")
	pullCmd.Flags().String("file-type", "", "File type of the files that do not set their own\n"+formatOptions(fileTypesOpt, 2, "unordered"))
	pullCmd.Flags().String("java-properties-encoding", "", "Character encoding for java_properties file type (default \"latin_1\")\n"+formatOptions(javaPropertiesEncodingOpt, 1, "unordered"))
	pullCmd.Flags().String("export-empty-as", "", "Export empty translations as (default \"empty\")\n"+formatOptions(exportEmptyAsOpt, 1, "unordered"))
	pullCmd.Flags().StringSlice("include-tags", []string{}, "List of tags to include in pull\nIf not set, all string keys will be considered for download")
	pullCmd.Flags().StringSlice("exclude-tags", []string{}, "List of tags to exclude from pull\nIf not set, all string keys will be considered for download")
	pullCmd.Flags().String("placeholder-format", "", "Placeholder syntax of the files, only for projects with universal placeholders and generic file types (json, java_properties, csv, xlsx, angular_xlf)\n"+formatOptions(placeholderFormatsOpt, 2, "unordered"))
}
