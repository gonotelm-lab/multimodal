package schema

type Language string

const (
	LanguageAuto    Language = ""
	LanguageChinese Language = "zh-CN"
	LanguageEnglish Language = "en-US"
)

type Request struct {
	Model       string
	Text        string
	Voice       string
	Language    Language
	Instruction string
}
