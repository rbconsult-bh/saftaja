package checkout

import (
	"encoding/json"
	"net/http"

	"golang.org/x/text/language"
)

var (
	LangEnglish = language.English
	LangArabic  = language.Arabic
)

var SupportedLanguages = []language.Tag{LangEnglish, LangArabic}
var languageMatcher = language.NewMatcher(SupportedLanguages)

func DetectLanguage(r *http.Request) language.Tag {
	if cookie, err := r.Cookie("lang"); err == nil {
		if tag, err := language.Parse(cookie.Value); err == nil {
			for _, supported := range SupportedLanguages {
				if tag == supported {
					return tag
				}
			}
		}
	}

	accept := r.Header.Get("Accept-Language")
	if accept == "" {
		return LangEnglish
	}
	tag, _ := language.MatchStrings(languageMatcher, accept)
	return tag
}

type LocalizedString map[language.Tag]string

func (ls LocalizedString) Get(tag language.Tag) string {
	if s, ok := ls[tag]; ok {
		return s
	}
	return ls[LangEnglish]
}

func (ls LocalizedString) ForRequest(r *http.Request) string {
	return ls.Get(DetectLanguage(r))
}

func (ls LocalizedString) MarshalJSON() ([]byte, error) {
	m := make(map[string]string, len(ls))
	for tag, s := range ls {
		m[tag.String()] = s
	}
	return json.Marshal(m)
}

func (ls *LocalizedString) UnmarshalJSON(data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	*ls = make(LocalizedString, len(m))
	for k, v := range m {
		tag, err := language.Parse(k)
		if err != nil {
			continue
		}
		(*ls)[tag] = v
	}
	return nil
}
