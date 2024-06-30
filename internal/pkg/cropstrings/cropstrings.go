package cropstrings

import "unicode/utf8"

func CropToLength(str string, length int) string {
	if utf8.RuneCountInString(str) <= length {
		return str
	}
	runes := []rune(str)
	runes = runes[:length]
	return string(runes)
}
