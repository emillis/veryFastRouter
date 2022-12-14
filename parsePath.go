package veryFastRouter

import "errors"

//===========[FUNCTIONALITY]====================================================================================================

//splitIntoSegments splits path and returns a slice of its values
func splitIntoSegments(path string) []segment {
	var buffer []segment
	var j int

	for i := len(path) - 1; i >= 0; i-- {
		if path[i] != 47 {
			continue
		}

		buffer = append(buffer, newSegment(path[i:]))
		path = path[:i]
		i = len(path)

		j++
	}

	return buffer
}

//fullPathCheck performs full check of the path
func fullPathCheck(path string) (string, error) {
	if path == "" {
		return path, errors.New("path supplied cannot be an empty string")
	}

	if path[0] != 47 {
		return path, errors.New("path must begin with \"/\"")
	}

	path = removeTrailingSlash(path)

	return path, nil
}

//removeTrailingSlash check for critical errors within the path supplied. Also, removes trailing "/" sign if present
func removeTrailingSlash(s string) string {
	if s[len(s)-1] == 47 && len(s) > 1 {
		return s[:len(s)-1]
	}

	return s
}
