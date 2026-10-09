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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/localizely/localizely-client-go"
)

const defaultApiBaseUrl = "https://api.localizely.com"

// apiClient is the generated client of the Localizely API (https://github.com/localizely/localizely-client-go),
// authenticated with the API token. Every call to the API goes through it.
type apiClient struct {
	*localizely.APIClient
	ctx context.Context
}

func newApiClient(apiToken string) *apiClient {
	baseUrl := strings.TrimRight(os.Getenv("LOCALIZELY_API_BASE_URL"), "/")
	if baseUrl == "" {
		baseUrl = defaultApiBaseUrl
	}

	cfg := localizely.NewConfiguration()
	cfg.Servers = localizely.ServerConfigurations{{URL: baseUrl}}
	cfg.UserAgent = "localizely-cli/" + Version
	cfg.HTTPClient = &http.Client{Timeout: 10 * time.Minute}
	ctx := context.WithValue(context.Background(), localizely.ContextAPIKeys, map[string]localizely.APIKey{"API auth": {Key: apiToken}})

	return &apiClient{APIClient: localizely.NewAPIClient(cfg), ctx: ctx}
}

// downloadParams are the parameters of the download section; the tags of an item override them for that file
type downloadParams struct {
	branch                 string
	fileType               string
	includeTags            []string
	excludeTags            []string
	exportEmptyAs          string
	javaPropertiesEncoding string
	placeholderFormat      string
}

// uploadParams are the parameters of the upload section; the tags of an item override them for that file
type uploadParams struct {
	branch    string
	overwrite bool
	// nil when not set: file formats with a translation state (String Catalogs) then take it per translation
	reviewed          *bool
	tagAdded          []string
	tagUpdated        []string
	tagRemoved        []string
	tagInFile         []string
	placeholderFormat string
}

// downloadFile pulls one localization file. A file with all locales, such as an Apple String Catalog, is pulled
// with every locale of the project.
func (api *apiClient) downloadFile(projectId string, file LocalizationFile, params downloadParams) ([]byte, error) {
	req := api.DownloadAPIAPI.GetLocalizationFile(api.ctx, projectId).Type_(resolveFileType(file, params.fileType))
	if !isMultiLocaleFile(file, params.fileType) {
		req = req.LangCodes(file.LocaleCode)
	}
	if params.branch != "" {
		req = req.Branch(params.branch)
	}
	if tags := resolveTags(file.IncludeTags, params.includeTags); len(tags) > 0 {
		req = req.IncludeTags(tags)
	}
	if tags := resolveTags(file.ExcludeTags, params.excludeTags); len(tags) > 0 {
		req = req.ExcludeTags(tags)
	}
	if params.exportEmptyAs != "" {
		req = req.ExportEmptyAs(params.exportEmptyAs)
	}
	if params.javaPropertiesEncoding != "" {
		req = req.JavaPropertiesEncoding(params.javaPropertiesEncoding)
	}
	if params.placeholderFormat != "" {
		req = req.PlaceholderFormat(params.placeholderFormat)
	}

	resp, err := req.Execute()
	if err != nil {
		return nil, describeError(resp, err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// uploadFile pushes one localization file for one language
func (api *apiClient) uploadFile(projectId string, file LocalizationFile, locale string, params uploadParams) error {
	f, err := os.Open(filepath.Clean(file.File))
	if err != nil {
		return err
	}
	defer f.Close()

	req := api.UploadAPIAPI.ImportLocalizationFile(api.ctx, projectId).LangCode(locale).File(f).Overwrite(params.overwrite)
	if params.reviewed != nil {
		req = req.Reviewed(*params.reviewed)
	}
	if params.branch != "" {
		req = req.Branch(params.branch)
	}
	if tags := resolveTags(file.TagAdded, params.tagAdded); len(tags) > 0 {
		req = req.TagAdded(tags)
	}
	if tags := resolveTags(file.TagUpdated, params.tagUpdated); len(tags) > 0 {
		req = req.TagUpdated(tags)
	}
	if tags := resolveTags(file.TagRemoved, params.tagRemoved); len(tags) > 0 {
		req = req.TagRemoved(tags)
	}
	if tags := resolveTags(file.TagInFile, params.tagInFile); len(tags) > 0 {
		req = req.TagInFile(tags)
	}
	if params.placeholderFormat != "" {
		req = req.PlaceholderFormat(params.placeholderFormat)
	}

	resp, err := req.Execute()

	return describeError(resp, err)
}

// projectLocales lists the locale codes of the project branch
func (api *apiClient) projectLocales(projectId string, branch string) ([]string, error) {
	req := api.TranslationStatusAPIAPI.GetTranslationStatus(api.ctx, projectId)
	if branch != "" {
		req = req.Branch(branch)
	}

	status, resp, err := req.Execute()
	if err != nil {
		return nil, describeError(resp, err)
	}

	locales := make([]string, 0, len(status.Languages))
	for _, language := range status.Languages {
		locales = append(locales, language.GetLangCode())
	}

	return locales, nil
}

// resolveTags is the list of an item when it sets one, otherwise the list of the section, without blank tags
func resolveTags(fileLevel []string, sectionLevel []string) []string {
	tags := []string{}
	for _, tag := range resolveList(fileLevel, sectionLevel) {
		if tag != "" {
			tags = append(tags, tag)
		}
	}

	return tags
}

// describeError describes a failed call with the HTTP status and the body the API returns with an error status,
// which carries the error code and message
func describeError(resp *http.Response, err error) error {
	var apiErr *localizely.GenericOpenAPIError
	if err == nil || resp == nil || !errors.As(err, &apiErr) {
		return err
	}

	if body := strings.TrimSpace(string(apiErr.Body())); body != "" {
		return fmt.Errorf("%s\n%s", resp.Status, body)
	}

	return errors.New(resp.Status)
}
