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

	"github.com/localizely/localizely-client-go"
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
		viper.BindPFlag("upload.files", cmd.Flags().Lookup("files"))
		viper.BindPFlag("upload.params.overwrite", cmd.Flags().Lookup("overwrite"))
		viper.BindPFlag("upload.params.reviewed", cmd.Flags().Lookup("reviewed"))
		viper.BindPFlag("upload.params.tag_added", cmd.Flags().Lookup("tag-added"))
		viper.BindPFlag("upload.params.tag_updated", cmd.Flags().Lookup("tag-updated"))
		viper.BindPFlag("upload.params.tag_removed", cmd.Flags().Lookup("tag-removed"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		apiToken := viper.GetString("api_token")
		projectId := viper.GetString("project_id")
		branch := viper.GetString("branch")
		files := viper.Get("upload.files")
		overwrite := viper.GetBool("upload.params.overwrite")
		reviewed := viper.GetBool("upload.params.reviewed")
		tagAdded := viper.GetStringSlice("upload.params.tag_added")
		tagUpdated := viper.GetStringSlice("upload.params.tag_updated")
		tagRemoved := viper.GetStringSlice("upload.params.tag_removed")

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

		err = validateFiles(localizationFiles, "push")
		checkError(err)

		filesMap := make(map[string]*os.File)
		for _, v := range localizationFiles {
			file, err := os.Open(filepath.Clean(v.file))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to open file '%s'\nError: %v\n", filepath.Clean(v.file), err)
				os.Exit(1)
			}

			filesMap[v.localeCode] = file
		}

		cfg := localizely.NewConfiguration()
		apiClient := localizely.NewAPIClient(cfg)

		ctx := context.WithValue(context.Background(), localizely.ContextAPIKeys, map[string]localizely.APIKey{"API auth": {Key: apiToken}})

		for _, localizationFile := range localizationFiles {
			file := filesMap[localizationFile.localeCode]

			resp, err := apiClient.UploadAPIApi.ImportLocalizationFile(ctx, projectId).Branch(branch).LangCode(localizationFile.localeCode).File(file).Overwrite(overwrite).Reviewed(reviewed).TagAdded(tagAdded).TagUpdated(tagUpdated).TagRemoved(tagRemoved).Execute()
			if err != nil {
				b, _ := io.ReadAll(resp.Body)
				jsonErr := string(b)
				fmt.Fprintf(os.Stderr, "Failed to push localization file %s to Localizely\nError: %s\n%s\n", file.Name(), err, jsonErr)
				os.Exit(1)
			}
		}

		fmt.Fprintln(os.Stdout, "Successfully pushed localization files to Localizely")
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)

	pushCmd.Flags().String("api-token", "", "API token\nYour API token from https://app.localizely.com/account")
	pushCmd.Flags().String("project-id", "", "Project ID\nYour project ID from https://app.localizely.com/projects")
	pushCmd.Flags().String("branch", "", "Branch name\nBranch in Localizely project to sync files with")
	pushCmd.Flags().StringToString("files", map[string]string{}, "List of localization files to push to Localizely\nExample:\n\t--files \"file[0]=lang/en_US.json\",\"locale_code[0]=en-US\"")
	pushCmd.Flags().Bool("overwrite", false, "Overwrite translations\nIf the translation in a given language should be overwritten with modified translation from uploading file")
	pushCmd.Flags().Bool("reviewed", false, "Mark translations as reviewed\nIf uploading translations, that are added, should be marked as Reviewed\nFor uploading translations that are only modified it will have effect only if overwrite is set to true")
	pushCmd.Flags().StringSlice("tag-added", []string{}, "List of tags to add to new translations from uploading file")
	pushCmd.Flags().StringSlice("tag-updated", []string{}, "List of tags to add to updated translations from uploading file")
	pushCmd.Flags().StringSlice("tag-removed", []string{}, "List of tags to add to removed translations from uploading file")
}
