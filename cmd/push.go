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
	"errors"
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

var pushCmd = &cobra.Command{
	Use:     "push",
	Short:   "Push localization files to Localizely",
	Example: "  localizely-cli push \\\n    --api-token 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \\\n    --project-id 01234567-abcd-abcd-abcd-0123456789ab \\\n    --files \"file[0]=lang/en.json\",\"locale_code[0]=en\",\"file[1]=lang/de_DE.json\",\"locale_code[1]=de-DE\" \\\n    --overwrite \\\n    --reviewed=false \\\n    --tag-added new,new-feat-x \\\n    --tag-updated updated,updated-feat-x \\\n    --tag-removed removed",
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

		err = pushLocalizationFiles(apiToken, projectId, branch, localizationFiles, overwrite, reviewed, tagAdded, tagUpdated, tagRemoved)
		checkError(err)

		color.Green("Successfully pushed data to Localizely")
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

func pushLocalizationFiles(apiToken string, projectId string, branch string, files []LocalizationFile, overwrite bool, reviewed bool, tagAdded []string, tagUpdated []string, tagRemoved []string) error {
	filesMap := make(map[string]*os.File)
	for _, v := range files {
		file, err := os.Open(filepath.Clean(v.File))
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to open file '%s'\nError: %v\n", filepath.Clean(v.File), err))
		}
		defer file.Close()
		filesMap[v.LocaleCode] = file
	}

	cfg := localizely.NewConfiguration()
	apiClient := localizely.NewAPIClient(cfg)
	ctx := context.WithValue(context.Background(), localizely.ContextAPIKeys, map[string]localizely.APIKey{"API auth": {Key: apiToken}})

	for _, v := range files {
		file := filesMap[v.LocaleCode]

		req := apiClient.UploadAPIApi.ImportLocalizationFile(ctx, projectId)
		req = req.LangCode(v.LocaleCode)
		req = req.File(file)
		req = req.Overwrite(overwrite)
		req = req.Reviewed(reviewed)
		if branch != "" {
			req = req.Branch(branch)
		}
		if len(tagAdded) > 0 {
			req = req.TagAdded(tagAdded)
		}
		if len(tagUpdated) > 0 {
			req = req.TagUpdated(tagUpdated)
		}
		if len(tagRemoved) > 0 {
			req = req.TagRemoved(tagRemoved)
		}

		resp, err := req.Execute()
		if err != nil {
			b, _ := io.ReadAll(resp.Body)
			jsonErr := string(b)
			return errors.New(fmt.Sprintf("Failed to push localization file '%s' to Localizely\nError: %v\n%s\n", file.Name(), err, jsonErr))
		}
		defer resp.Body.Close()
	}

	return nil
}
