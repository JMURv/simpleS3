package slugify

import (
	"path/filepath"
	"strings"
	"unicode"
)

var transliteration = map[rune]string{
	'А': "A", 'Б': "B", 'В': "V", 'Г': "G", 'Д': "D", 'Е': "E", 'Ё': "YO", 'Ж': "ZH", 'З': "Z", 'И': "I",
	'Й': "Y", 'К': "K", 'Л': "L", 'М': "M", 'Н': "N", 'О': "O", 'П': "P", 'Р': "R", 'С': "S", 'Т': "T",
	'У': "U", 'Ф': "F", 'Х': "KH", 'Ц': "TS", 'Ч': "CH", 'Ш': "SH", 'Щ': "SHCH", 'Ь': "'", 'Ы': "Y", 'Ъ': "",
	'Э': "E", 'Ю': "YU", 'Я': "YA", 'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p",
	'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
	'ь': "'", 'ы': "y", 'ъ': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

func transliterate(cyrillic string) string {
	var result strings.Builder
	for _, r := range cyrillic {
		if replacement, ok := transliteration[r]; ok {
			result.WriteString(replacement)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func splitFileNameAndExt(s string) (string, string) {
	ext := filepath.Ext(s)
	name := strings.TrimSuffix(s, ext)
	return name, strings.TrimPrefix(ext, ".")
}

func Filename(s string) string {
	s = transliterate(strings.TrimSpace(s))

	name, ext := splitFileNameAndExt(s)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ToLower(name)

	var result strings.Builder
	hyphenLast := false
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
			hyphenLast = false
		} else if r == '-' && !hyphenLast {
			result.WriteRune(r)
			hyphenLast = true
		}
	}

	if ext != "" {
		result.WriteRune('.')
		result.WriteString(ext)
	}
	return result.String()
}
