package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUploadFile_sendsFileAndParameters(t *testing.T) {
	var got struct {
		path, token, agent, filename, content string
		query                                 url.Values
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.query = r.URL.Query()
		got.token = r.Header.Get("X-Api-Token")
		got.agent = r.Header.Get("User-Agent")
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("missing file part: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		content, _ := io.ReadAll(file)
		got.filename = header.Filename
		got.content = string(content)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "strings.xml")
	if err := os.WriteFile(path, []byte("<resources/>"), 0666); err != nil {
		t.Fatal(err)
	}
	params := url.Values{}
	params.Set("lang_code", "en-US")
	params.Add("tag_in_file", "android")
	params.Add("tag_in_file", "mobile")
	params.Set("placeholder_format", "printf_java")

	client := &apiClient{baseUrl: server.URL, apiToken: "token", client: server.Client()}
	if err := client.uploadFile("p1", params, path); err != nil {
		t.Fatal(err)
	}

	if got.path != "/v1/projects/p1/files/upload" || got.token != "token" || got.agent != "localizely-cli/"+Version {
		t.Fatalf("request %+v", got)
	}
	if got.filename != "strings.xml" || got.content != "<resources/>" {
		t.Fatalf("file %q %q", got.filename, got.content)
	}
	if !reflect.DeepEqual(got.query["tag_in_file"], []string{"android", "mobile"}) || got.query.Get("placeholder_format") != "printf_java" || got.query.Get("lang_code") != "en-US" {
		t.Fatalf("query %v", got.query)
	}
}

func TestDownloadFile_returnsBodyAndReportsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("type") == "ios_xcstrings" && r.URL.Query().Get("lang_codes") == "" {
			w.Write([]byte(`{"sourceLanguage":"en"}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorCode":"bad_request"}`))
	}))
	defer server.Close()
	client := &apiClient{baseUrl: server.URL, apiToken: "token", client: server.Client()}

	params := url.Values{}
	params.Set("type", "ios_xcstrings")
	body, err := client.downloadFile("p1", params)
	if err != nil || string(body) != `{"sourceLanguage":"en"}` {
		t.Fatalf("body %q err %v", body, err)
	}

	params.Set("lang_codes", "xx")
	_, err = client.downloadFile("p1", params)
	apiErr, ok := err.(*apiError)
	if !ok || apiErr.status != http.StatusBadRequest || apiErr.body != `{"errorCode":"bad_request"}` {
		t.Fatalf("expected the API error with its body, got %v", err)
	}
}

func TestProjectLocales(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/p1/status" || r.URL.Query().Get("branch") != "main" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"strings":3,"languages":[{"langCode":"en-US","langName":"English"},{"langCode":"de","langName":"German"}]}`))
	}))
	defer server.Close()
	client := &apiClient{baseUrl: server.URL, apiToken: "token", client: server.Client()}

	locales, err := client.projectLocales("p1", "main")
	if err != nil || !reflect.DeepEqual(locales, []string{"en-US", "de"}) {
		t.Fatalf("locales %v err %v", locales, err)
	}
}
