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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// newTestApiClient is an API client pointed at a test server through LOCALIZELY_API_BASE_URL
func newTestApiClient(t *testing.T, handler http.HandlerFunc) *apiClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Setenv("LOCALIZELY_API_BASE_URL", server.URL+"/")

	return newApiClient("token")
}

func TestUploadFile_sendsFileAndParameters(t *testing.T) {
	var got struct {
		path, token, agent, filename, content string
		query                                 url.Values
	}
	api := newTestApiClient(t, func(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	path := filepath.Join(t.TempDir(), "strings.xml")
	if err := os.WriteFile(path, []byte("<resources/>"), 0666); err != nil {
		t.Fatal(err)
	}
	// The item's own tag_in_file overrides the section's; blank tags are dropped; reviewed is left out when not set
	file := LocalizationFile{File: path, LocaleCode: "en-US", TagInFile: []string{"android", "mobile"}}
	params := uploadParams{branch: "main", overwrite: true, tagAdded: []string{"new", ""}, tagInFile: []string{"web"}, placeholderFormat: "printf_java"}
	if err := api.uploadFile("p1", file, "en-US", params); err != nil {
		t.Fatal(err)
	}

	if got.path != "/v1/projects/p1/files/upload" || got.token != "token" || got.agent != "localizely-cli/"+Version {
		t.Fatalf("request %+v", got)
	}
	if got.filename != "strings.xml" || got.content != "<resources/>" {
		t.Fatalf("file %q %q", got.filename, got.content)
	}
	expected := url.Values{"lang_code": {"en-US"}, "branch": {"main"}, "overwrite": {"true"}, "tag_added": {"new"}, "tag_in_file": {"android", "mobile"}, "placeholder_format": {"printf_java"}}
	if !reflect.DeepEqual(got.query, expected) {
		t.Fatalf("query %v", got.query)
	}
}

func TestUploadFile_sendsReviewedWhenSet(t *testing.T) {
	var got url.Values
	api := newTestApiClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	})

	path := filepath.Join(t.TempDir(), "en.json")
	if err := os.WriteFile(path, []byte("{}"), 0666); err != nil {
		t.Fatal(err)
	}
	reviewed := false
	if err := api.uploadFile("p1", LocalizationFile{File: path, LocaleCode: "en"}, "en", uploadParams{reviewed: &reviewed}); err != nil {
		t.Fatal(err)
	}

	expected := url.Values{"lang_code": {"en"}, "overwrite": {"false"}, "reviewed": {"false"}}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("query %v", got)
	}
}

func TestDownloadFile_returnsBodyAndReportsErrors(t *testing.T) {
	var got url.Values
	api := newTestApiClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		if r.URL.Path == "/v1/projects/p1/files/download" && got.Get("type") == "ios_xcstrings" {
			w.Write([]byte(`{"sourceLanguage":"en"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorCode":"bad_request","errorMessage":"Unknown language"}`))
	})

	// A file with all locales is pulled without lang_codes, and its own include_tags override the section's
	catalog := LocalizationFile{File: "App/Localizable.xcstrings", FileType: "ios_xcstrings", IncludeTags: []string{"ios"}}
	params := downloadParams{branch: "main", fileType: "json", includeTags: []string{"web"}, excludeTags: []string{"removed"}, exportEmptyAs: "skip", placeholderFormat: "i18next"}
	body, err := api.downloadFile("p1", catalog, params)
	if err != nil || string(body) != `{"sourceLanguage":"en"}` {
		t.Fatalf("body %q err %v", body, err)
	}
	expected := url.Values{"type": {"ios_xcstrings"}, "branch": {"main"}, "include_tags": {"ios"}, "exclude_tags": {"removed"}, "export_empty_as": {"skip"}, "placeholder_format": {"i18next"}}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("query %v", got)
	}

	_, err = api.downloadFile("p1", LocalizationFile{File: "lang/xx.json", LocaleCode: "xx"}, params)
	if err == nil || !strings.HasPrefix(err.Error(), "400 Bad Request\n") || !strings.Contains(err.Error(), `"errorCode":"bad_request"`) {
		t.Fatalf("expected the API error with its body, got %v", err)
	}
	if got.Get("type") != "json" || got.Get("lang_codes") != "xx" {
		t.Fatalf("query %v", got)
	}
}

func TestProjectLocales(t *testing.T) {
	api := newTestApiClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/p1/status" || r.URL.Query().Get("branch") != "main" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"strings":3,"languages":[{"langCode":"en-US","langName":"English"},{"langCode":"de","langName":"German"}]}`))
	})

	locales, err := api.projectLocales("p1", "main")
	if err != nil || !reflect.DeepEqual(locales, []string{"en-US", "de"}) {
		t.Fatalf("locales %v err %v", locales, err)
	}
}
