// Package i18n provides a global localizer initialized once at startup.
package i18n

import (
	"log/slog"
	"smart-pc-agent/data/messages"

	"github.com/BurntSushi/toml"
	golocale "github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var localizer *i18n.Localizer

// Init loads message files and creates a global localizer.
// Must be called once before any T* function is used.
func Init(log *slog.Logger) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	for _, f := range []string{
		"active.en.toml",
		"active.ru.toml",
	} {
		if _, err := bundle.LoadMessageFileFS(messages.FS, f); err != nil {
			log.Error(
				"i18n: failed to load message file",
				slog.String("file", f),
				slog.Any("err", err),
			)
		}
	}

	langs := systemLanguages(log)
	localizer = i18n.NewLocalizer(bundle, langs...)
}

// systemLanguages returns BCP-47 language tags ordered by user preference.
// Falls back to English on any error.
func systemLanguages(log *slog.Logger) []string {
	locales, err := golocale.GetLocales()
	if err != nil || len(locales) == 0 {
		if err != nil {
			log.Warn(
				"i18n: could not detect system locale, falling back to English",
				slog.Any("err", err),
			)
		}
		return []string{language.English.String()}
	}
	return locales
}

// T returns the localized string for the given message ID.
// Falls back to the defaultMsg when the ID is missing from all catalogs.
func T(id, defaultMsg string) string {
	return TData(id, defaultMsg, nil)
}

// TData is like T but also interpolates TemplateData.
func TData(id, defaultMsg string, data map[string]any) string {
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID:    id,
			Other: defaultMsg,
		},
		TemplateData: data,
	})
	if err != nil {
		return defaultMsg
	}
	return msg
}
