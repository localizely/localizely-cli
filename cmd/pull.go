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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/fatih/color"
	"github.com/localizely/localizely-client-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pullCmd = &cobra.Command{
	Use:     "pull",
	Short:   "Pull localization files from Localizely",
	Example: "  localizely-cli pull \\\n    --api-token 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \\\n    --project-id 01234567-abcd-abcd-abcd-0123456789ab \\\n    --file-type json \\\n    --files \"file[0]=lang/en.json\",\"locale_code[0]=en\",\"file[1]=lang/de_DE.json\",\"locale_code[1]=de-DE\" \\\n    --export-empty-as empty \\\n    --include-tags new,updated \\\n    --exclude-tags removed",
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

		err = validateFiles(localizationFiles, "pull")
		checkError(err)

		err = validateExportEmptyAs(exportEmptyAs)
		checkError(err)

		err = validateJavaPropertiesEncoding(javaPropertiesEncoding)
		checkError(err)

		cfg := localizely.NewConfiguration()
		apiClient := localizely.NewAPIClient(cfg)
		ctx := context.WithValue(context.Background(), localizely.ContextAPIKeys, map[string]localizely.APIKey{"API auth": {Key: apiToken}})

		for _, localizationFile := range localizationFiles {
			req := apiClient.DownloadAPIApi.GetLocalizationFile(ctx, projectId)
			req = req.LangCodes(localizationFile.LocaleCode)
			req = req.Type_(fileType)
			if branch != "" {
				req = req.Branch(branch)
			}
			if len(includeTags) > 0 {
				req = req.IncludeTags(includeTags)
			}
			if len(excludeTags) > 0 {
				req = req.ExcludeTags(excludeTags)
			}
			if exportEmptyAs != "" {
				req = req.ExportEmptyAs(exportEmptyAs)
			}
			if javaPropertiesEncoding != "" {
				req = req.JavaPropertiesEncoding(javaPropertiesEncoding)
			}

			resp, err := req.Execute()
			if err != nil {
				b, _ := io.ReadAll(resp.Body)
				jsonErr := string(b)
				fmt.Fprintf(os.Stderr, "Failed to pull data from Localizely\nError: %v\n%s\n", err, jsonErr)
				os.Exit(1)
			}

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read response from the server\nError: %v\n", err)
				os.Exit(1)
			}

			err = os.MkdirAll(filepath.Dir(localizationFile.File), 0777)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create directory '%s'\nError: %v\n", filepath.Dir(localizationFile.File), err)
				os.Exit(1)
			}

			err = os.WriteFile(filepath.Clean(localizationFile.File), b, 0666)
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
	pullCmd.Flags().StringToString("files", map[string]string{}, "List of localization files to pull from Localizely\nExample:\n\t--files \"file[0]=lang/en_US.json\",\"locale_code[0]=en-US\"")
	pullCmd.Flags().String("file-type", "", "File type\n"+formatOptions(fileTypesOpt, 2, "unordered"))
	pullCmd.Flags().String("java-properties-encoding", "", "Character encoding for java_properties file type (default \"latin_1\")\n"+formatOptions(javaPropertiesEncodingOpt, 1, "unordered"))
	pullCmd.Flags().String("export-empty-as", "", "Export empty translations as (default \"empty\")\n"+formatOptions(exportEmptyAsOpt, 1, "unordered"))
	pullCmd.Flags().StringSlice("include-tags", []string{}, "List of tags to include in pull\nIf not set, all string keys will be considered for download")
	pullCmd.Flags().StringSlice("exclude-tags", []string{}, "List of tags to exclude from pull\nIf not set, all string keys will be considered for download")
}
