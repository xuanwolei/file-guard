package services

import "unicode/utf8"

func truncateUTF8(text string, maxBytes int) string {
	if maxBytes <= 0 || len(text) <= maxBytes {
		return text
	}
	end := 0
	for end < len(text) {
		_, size := utf8.DecodeRuneInString(text[end:])
		if end+size > maxBytes {
			break
		}
		end += size
	}
	return text[:end]
}

func truncateWithSuffix(text string, maxBytes int, suffix string) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(text) <= maxBytes {
		return text
	}
	if len(suffix) >= maxBytes {
		return truncateUTF8(suffix, maxBytes)
	}
	return truncateUTF8(text, maxBytes-len(suffix)) + suffix
}

func addTruncationSuffix(text string, maxBytes int, suffix string) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(suffix) >= maxBytes {
		return truncateUTF8(suffix, maxBytes)
	}
	return truncateUTF8(text, maxBytes-len(suffix)) + suffix
}
