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
	"reflect"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push localization files to Localizely",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Bind flags only if the command is executed (fixes issue with global viper and the same flag names in multiple cobra commands)
		// More info: https://github.com/spf13/viper/issues/233#issuecomment-386791444
		viper.BindPFlag("api_token", cmd.Flags().Lookup("api-token"))
		viper.BindPFlag("project_id", cmd.Flags().Lookup("project-id"))
		viper.BindPFlag("branch", cmd.Flags().Lookup("branch"))
		viper.BindPFlag("file_type", cmd.Flags().Lookup("file-type"))
		viper.BindPFlag("upload.files", cmd.Flags().Lookup("files"))
		viper.BindPFlag("upload.params.overwrite", cmd.Flags().Lookup("overwrite"))
		viper.BindPFlag("upload.params.reviewed", cmd.Flags().Lookup("reviewed"))
		viper.BindPFlag("upload.params.tag_added", cmd.Flags().Lookup("tag-added"))
		viper.BindPFlag("upload.params.tag_updated", cmd.Flags().Lookup("tag-updated"))
		viper.BindPFlag("upload.params.tag_removed", cmd.Flags().Lookup("tag-removed"))
		viper.BindPFlag("upload.params.tag_in_file", cmd.Flags().Lookup("tag-in-file"))
		viper.BindPFlag("upload.params.placeholder_format", cmd.Flags().Lookup("placeholder-format"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		apiToken := viper.GetString("api_token")
		projectId := viper.GetString("project_id")
		branch := viper.GetString("branch")
		fileType := viper.GetString("file_type")
		files := viper.Get("upload.files")
		overwrite := viper.GetBool("upload.params.overwrite")
		// Left out unless set: file formats with a translation state (String Catalogs) then take it per translation
		reviewedSet := viper.IsSet("upload.params.reviewed")
		reviewed := viper.GetBool("upload.params.reviewed")
		tagAdded := viper.GetStringSlice("upload.params.tag_added")
		tagUpdated := viper.GetStringSlice("upload.params.tag_updated")
		tagRemoved := viper.GetStringSlice("upload.params.tag_removed")
		tagInFile := viper.GetStringSlice("upload.params.tag_in_file")
		placeholderFormat := viper.GetString("upload.params.placeholder_format")

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

		err = validateFiles(localizationFiles, fileType, "push")
		checkError(err)

		err = validatePlaceholderFormat(placeholderFormat)
		checkError(err)

		for _, localizationFile := range localizationFiles {
			if _, err := os.Stat(localizationFile.File); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open file '%s'\nError: %v\n", localizationFile.File, err)
				os.Exit(1)
			}
		}

		api := newApiClient(apiToken)
		var projectLocales []string

		for _, localizationFile := range localizationFiles {
			params := url.Values{}
			params.Set("overwrite", strconv.FormatBool(overwrite))
			if reviewedSet {
				params.Set("reviewed", strconv.FormatBool(reviewed))
			}
			if branch != "" {
				params.Set("branch", branch)
			}
			// File-level tags override the section parameters for this file
			addTags(params, "tag_added", resolveList(localizationFile.TagAdded, tagAdded))
			addTags(params, "tag_updated", resolveList(localizationFile.TagUpdated, tagUpdated))
			addTags(params, "tag_removed", resolveList(localizationFile.TagRemoved, tagRemoved))
			addTags(params, "tag_in_file", resolveList(localizationFile.TagInFile, tagInFile))
			if placeholderFormat != "" {
				params.Set("placeholder_format", placeholderFormat)
			}

			if !isMultiLocaleFile(localizationFile, fileType) {
				params.Set("lang_code", localizationFile.LocaleCode)
				pushFile(api, projectId, params, localizationFile.File)
				continue
			}

			// A file with all locales, such as an Apple String Catalog, is pushed once per language of the project
			if projectLocales == nil {
				projectLocales, err = api.projectLocales(projectId, branch)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to read the languages of the project from Localizely\nError: %v\n", err)
					os.Exit(1)
				}
			}
			fileLocales, err := catalogLocales(localizationFile.File)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read the languages of the file '%s'\nError: %v\n", localizationFile.File, err)
				os.Exit(1)
			}
			matched, skipped := matchLocales(fileLocales, projectLocales)
			if len(matched) == 0 {
				fmt.Fprintf(os.Stderr, "None of the languages of the file '%s' (%s) exists in the project\n", localizationFile.File, strings.Join(fileLocales, ", "))
				os.Exit(1)
			}
			for _, locale := range matched {
				params.Set("lang_code", locale)
				pushFile(api, projectId, params, localizationFile.File)
			}
			fmt.Fprintf(os.Stderr, "Pushed '%s' for %s\n", localizationFile.File, strings.Join(matched, ", "))
			if len(skipped) > 0 {
				fmt.Fprintf(os.Stderr, "Skipped languages of '%s' that the project does not have: %s\n", localizationFile.File, strings.Join(skipped, ", "))
			}
		}

		color.Green("Successfully pushed data to Localizely")
	},
}

func addTags(params url.Values, name string, tags []string) {
	for _, tag := range tags {
		if tag != "" {
			params.Add(name, tag)
		}
	}
}

func pushFile(api *apiClient, projectId string, params url.Values, file string) {
	if err := api.uploadFile(projectId, params, file); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to push localization file '%s' to Localizely\nError: %v\n", file, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(pushCmd)

	pushCmd.Flags().String("api-token", "", "API token\nYour API token from https://app.localizely.com/account")
	pushCmd.Flags().String("project-id", "", "Project ID\nYour project ID from https://app.localizely.com/projects")
	pushCmd.Flags().String("branch", "", "Branch name\nBranch in Localizely project to sync files with")
	pushCmd.Flags().String("file-type", "", "File type, needed only for files that hold all locales (ios_xcstrings)\n"+formatOptions(fileTypesOpt, 2, "unordered"))
	pushCmd.Flags().StringToString("files", map[string]string{}, "List of localization files to push to Localizely\nExample:\n\t--files \"file[0]=lang/en_US.json\",\"locale_code[0]=en-US\"\nA file that holds all locales takes file_type[i] instead of locale_code[i]:\n\t--files \"file[0]=App/Localizable.xcstrings\",\"file_type[0]=ios_xcstrings\"")
	pushCmd.Flags().Bool("overwrite", false, "Overwrite translations\nIf the translation in a given language should be overwritten with modified translation from uploading file")
	pushCmd.Flags().Bool("reviewed", false, "Mark translations as reviewed\nIf uploading translations, that are added, should be marked as Reviewed\nFor uploading translations that are only modified it will have effect only if overwrite is set to true\nWhen not set, String Catalogs take the review state of each translation from the file")
	pushCmd.Flags().StringSlice("tag-added", []string{}, "List of tags to add to new translations from uploading file")
	pushCmd.Flags().StringSlice("tag-updated", []string{}, "List of tags to add to updated translations from uploading file")
	pushCmd.Flags().StringSlice("tag-removed", []string{}, "List of tags to add to removed translations from uploading file")
	pushCmd.Flags().StringSlice("tag-in-file", []string{}, "List of tags that mirror the file: added to every string key in the uploading file and removed from string keys that are not in it")
	pushCmd.Flags().String("placeholder-format", "", "Placeholder syntax of the file, only for projects with universal placeholders and generic file types (json, java_properties, csv, xlsx, angular_xlf)\n"+formatOptions(placeholderFormatsOpt, 2, "unordered"))
}
