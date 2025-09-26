package utils

// UniqueStrings returns a new slice containing only the unique strings from the input slice.
// The order of elements in the returned slice is not guaranteed to be the same as the input.
func UniqueStrings(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}

	// Use a map to track unique strings.
	seen := make(map[string]struct{})
	result := []string{}

	for _, str := range input {
		if _, ok := seen[str]; !ok {
			seen[str] = struct{}{}
			result = append(result, str)
		}
	}

	return result
}

// ReverseStrings reverses a slice of strings in place.
func ReverseStrings(s []string) []string {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

// ChunkText splits a text into chunks of a given size.
func ChunkText(text string, chunkSize int) []string {
	var chunks []string
	for i := 0; i < len(text); i += chunkSize {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
	}
	return chunks
}

// FirstNotEmpty returns the first non-empty string from the input slice.
func FirstNotEmpty(strs ...string) string {
	for _, str := range strs {
		if str != "" {
			return str
		}
	}
	return ""
}
