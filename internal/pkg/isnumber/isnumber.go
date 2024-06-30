package isnumber

import (
	"strconv"
	"unicode"
)

func IsNumber(num string) bool {
	for _, elem := range num {
		if !unicode.IsDigit(elem) {
			return false
		}
	}
	return true
}

func TryConvertToInt(num string) int {
	toRet := -1
	if num != "" && IsNumber(num) {
		toRet, _ = strconv.Atoi(num)
	}
	return toRet
}
