package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// catalogLocales lists the languages an Apple String Catalog holds: its source language and every language that has
// localizations. A catalog is pushed once per language, since an upload takes one language of the file.
func catalogLocales(path string) ([]string, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var catalog struct {
		SourceLanguage string `json:"sourceLanguage"`
		Strings        map[string]struct {
			Localizations map[string]json.RawMessage `json:"localizations"`
		} `json:"strings"`
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}

	set := map[string]bool{}
	if catalog.SourceLanguage != "" {
		set[catalog.SourceLanguage] = true
	}
	for _, entry := range catalog.Strings {
		for locale := range entry.Localizations {
			set[locale] = true
		}
	}

	locales := make([]string, 0, len(set))
	for locale := range set {
		locales = append(locales, locale)
	}
	sort.Strings(locales)

	return locales, nil
}

// normalizeLocale makes locale codes comparable the way Localizely matches them: case does not matter, and the
// underscore and dash notations are equal
func normalizeLocale(locale string) string {
	return strings.ToLower(strings.ReplaceAll(locale, "_", "-"))
}

// matchLocales pairs the languages of a file with the locales of the project: the project's spelling is used for
// the API call, and languages the project does not have are reported as skipped
func matchLocales(fileLocales []string, projectLocales []string) (matched []string, skipped []string) {
	byNormalized := map[string]string{}
	for _, locale := range projectLocales {
		byNormalized[normalizeLocale(locale)] = locale
	}
	for _, locale := range fileLocales {
		if projectLocale, ok := byNormalized[normalizeLocale(locale)]; ok {
			matched = append(matched, projectLocale)
		} else {
			skipped = append(skipped, locale)
		}
	}

	return matched, skipped
}
