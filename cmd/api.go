package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultApiBaseUrl = "https://api.localizely.com"

// apiClient talks to the Localizely REST API (https://api.localizely.com/swagger-ui/index.html)
type apiClient struct {
	baseUrl  string
	apiToken string
	client   *http.Client
}

func newApiClient(apiToken string) *apiClient {
	baseUrl := strings.TrimRight(os.Getenv("LOCALIZELY_API_BASE_URL"), "/")
	if baseUrl == "" {
		baseUrl = defaultApiBaseUrl
	}

	return &apiClient{
		baseUrl:  baseUrl,
		apiToken: apiToken,
		client:   &http.Client{Timeout: 10 * time.Minute},
	}
}

// apiError carries the JSON body the API returns with an error status
type apiError struct {
	status int
	body   string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("HTTP %d\n%s", e.status, strings.TrimSpace(e.body))
}

func (c *apiClient) send(req *http.Request) ([]byte, error) {
	req.Header.Set("X-Api-Token", c.apiToken)
	req.Header.Set("User-Agent", "localizely-cli/"+Version)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &apiError{status: resp.StatusCode, body: string(body)}
	}

	return body, nil
}

func (c *apiClient) projectUrl(projectId string, path string, params url.Values) string {
	u := c.baseUrl + "/v1/projects/" + url.PathEscape(projectId) + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	return u
}

// uploadFile pushes one localization file with the given query parameters
func (c *apiClient) uploadFile(projectId string, params url.Values, filePath string) error {
	file, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.projectUrl(projectId, "/files/upload", params), &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err = c.send(req)

	return err
}

// downloadFile pulls one localization file with the given query parameters
func (c *apiClient) downloadFile(projectId string, params url.Values) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.projectUrl(projectId, "/files/download", params), nil)
	if err != nil {
		return nil, err
	}

	return c.send(req)
}

// projectLocales lists the locale codes of the project branch
func (c *apiClient) projectLocales(projectId string, branch string) ([]string, error) {
	params := url.Values{}
	if branch != "" {
		params.Set("branch", branch)
	}

	req, err := http.NewRequest(http.MethodGet, c.projectUrl(projectId, "/status", params), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.send(req)
	if err != nil {
		return nil, err
	}

	var status struct {
		Languages []struct {
			LangCode string `json:"langCode"`
		} `json:"languages"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("unexpected response of the translation status endpoint: %w", err)
	}

	locales := make([]string, 0, len(status.Languages))
	for _, language := range status.Languages {
		locales = append(locales, language.LangCode)
	}

	return locales, nil
}
