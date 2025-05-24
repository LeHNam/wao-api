package i18nService

import (
	"context"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var bundle *i18n.Bundle
var bundleOnce sync.Once

func NewI18nService() *i18n.Bundle {
	bundleOnce.Do(func() {
		initBundle := i18n.NewBundle(language.English)
		initBundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
		bundle = initBundle
	})

	return bundle
}

func LocalizeMessageID(ctx context.Context, messageID string, templateData *map[string]interface{}) string {
	lang := ctx.Value("X-LANG").(string)
	if lang == "" {
		lang = "en"
	}
	bundle := NewI18nService()
	localizer := i18n.NewLocalizer(bundle, lang)
	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		return messageID
	}
	return msg
}
