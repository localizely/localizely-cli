package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestConvertFilesConfig_readsFileLevelOptions(t *testing.T) {
	var files []LocalizationFile
	convertFilesConfigToLocalizationFiles([]interface{}{
		map[string]interface{}{"file": "android/values/strings.xml", "locale_code": "en", "file_type": "android_xml", "tag_in_file": []interface{}{"android"}},
		map[string]interface{}{"file": "ios/Localizable.xcstrings", "file_type": "ios_xcstrings", "include_tags": []interface{}{"ios"}, "exclude_tags": []interface{}{}},
		map[string]interface{}{"file": "lang/de.json", "locale_code": "de"},
	}, &files)

	expected := []LocalizationFile{
		{File: "android/values/strings.xml", LocaleCode: "en", FileType: "android_xml", TagInFile: []string{"android"}},
		{File: "ios/Localizable.xcstrings", FileType: "ios_xcstrings", IncludeTags: []string{"ios"}, ExcludeTags: []string{}},
		{File: "lang/de.json", LocaleCode: "de"},
	}
	if !reflect.DeepEqual(files, expected) {
		t.Fatalf("got %#v", files)
	}
	if files[2].TagInFile != nil {
		t.Fatalf("an absent list must stay nil so that the section parameters apply")
	}
}

func TestConvertFilesFlag_readsFileTypeAndKeepsOrder(t *testing.T) {
	var files []LocalizationFile
	convertFilesFlagToLocalizationFiles(map[string]interface{}{
		"file[1]": "lang/de.json", "locale_code[1]": "de",
		"file[0]": "lang/en.json", "locale_code[0]": "en",
		"file[2]": "App/Localizable.xcstrings", "file_type[2]": "ios_xcstrings",
	}, &files)

	expected := []LocalizationFile{
		{File: "lang/en.json", LocaleCode: "en"},
		{File: "lang/de.json", LocaleCode: "de"},
		{File: "App/Localizable.xcstrings", FileType: "ios_xcstrings"},
	}
	if !reflect.DeepEqual(files, expected) {
		t.Fatalf("got %#v", files)
	}
}

func TestValidateFiles(t *testing.T) {
	cases := []struct {
		name     string
		files    []LocalizationFile
		fileType string
		command  string
		wantErr  string
	}{
		{"single locale file with top-level type", []LocalizationFile{{File: "a.json", LocaleCode: "en"}}, "json", "pull", ""},
		{"file-level type replaces the top-level one", []LocalizationFile{{File: "a.xml", LocaleCode: "en", FileType: "android_xml"}}, "", "pull", ""},
		{"pull needs a type for every file", []LocalizationFile{{File: "a.json", LocaleCode: "en"}}, "", "pull", "file type was not provided"},
		{"push reads the format from the file", []LocalizationFile{{File: "a.json", LocaleCode: "en"}}, "", "push", ""},
		{"invalid file-level type", []LocalizationFile{{File: "a.json", LocaleCode: "en", FileType: "xls"}}, "", "pull", "is invalid"},
		{"catalog with a locale code", []LocalizationFile{{File: "A.xcstrings", LocaleCode: "en", FileType: "ios_xcstrings"}}, "", "pull", "must be omitted"},
		{"catalog by extension on push", []LocalizationFile{{File: "A.xcstrings"}}, "", "push", ""},
		{"catalog by top-level type on pull", []LocalizationFile{{File: "A.xcstrings"}}, "ios_xcstrings", "pull", ""},
		{"missing locale code", []LocalizationFile{{File: "a.json"}}, "json", "pull", "locale_code"},
		{"listed twice", []LocalizationFile{{File: "a.json", LocaleCode: "en"}, {File: "a.json", LocaleCode: "de"}}, "json", "pull", "more than once"},
		{"no files", nil, "json", "pull", "was not provided"},
	}
	for _, c := range cases {
		err := validateFiles(c.files, c.fileType, c.command)
		if c.wantErr == "" && err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
		}
		if c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)) {
			t.Errorf("%s: expected error containing %q, got %v", c.name, c.wantErr, err)
		}
	}
}

func TestValidatePlaceholderFormat(t *testing.T) {
	if err := validatePlaceholderFormat(""); err != nil {
		t.Fatal(err)
	}
	if err := validatePlaceholderFormat("i18next"); err != nil {
		t.Fatal(err)
	}
	if err := validatePlaceholderFormat("printf"); err == nil {
		t.Fatal("expected an error for an unknown placeholder format")
	}
}

func TestResolveList(t *testing.T) {
	section := []string{"android"}
	if got := resolveList(nil, section); !reflect.DeepEqual(got, section) {
		t.Fatalf("an unset file-level list must fall back to the section, got %v", got)
	}
	if got := resolveList([]string{}, section); len(got) != 0 {
		t.Fatalf("an empty file-level list must mean no tags, got %v", got)
	}
}

func TestCatalogLocales_andMatching(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Localizable.xcstrings")
	catalog := `{
  "sourceLanguage" : "en",
  "strings" : {
    "greeting" : { "localizations" : { "de" : { "stringUnit" : { "state" : "translated", "value" : "Hallo" } }, "pt_BR" : { "stringUnit" : { "state" : "new", "value" : "Olá" } } } },
    "Localizely" : { "shouldTranslate" : false }
  },
  "version" : "1.0"
}`
	if err := os.WriteFile(path, []byte(catalog), 0666); err != nil {
		t.Fatal(err)
	}

	locales, err := catalogLocales(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(locales, []string{"de", "en", "pt_BR"}) {
		t.Fatalf("got %v", locales)
	}

	matched, skipped := matchLocales(locales, []string{"en-US", "pt-BR", "de"})
	if !reflect.DeepEqual(matched, []string{"de", "pt-BR"}) || !reflect.DeepEqual(skipped, []string{"en"}) {
		t.Fatalf("matched %v skipped %v", matched, skipped)
	}
}
