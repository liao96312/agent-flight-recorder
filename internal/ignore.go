package afr

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type ignorePattern struct {
	value  string
	isGlob bool
}

type afrIgnore struct {
	patterns []ignorePattern
}

func loadAFRIgnore(root string) (afrIgnore, error) {
	file, err := os.Open(filepath.Join(root, ".afrignore"))
	if os.IsNotExist(err) {
		return afrIgnore{}, nil
	}
	if err != nil {
		return afrIgnore{}, fmt.Errorf("read .afrignore: %w", err)
	}
	defer file.Close()

	ignore := afrIgnore{}
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		pattern := strings.TrimSpace(scanner.Text())
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}
		pattern = strings.ReplaceAll(pattern, `\`, "/")
		pattern = strings.TrimPrefix(pattern, "./")
		pattern = strings.TrimPrefix(pattern, "/")
		pattern = strings.TrimSuffix(pattern, "/")
		if pattern == "" || pattern == ".." || strings.HasPrefix(pattern, "../") {
			return afrIgnore{}, fmt.Errorf("invalid .afrignore pattern on line %d", lineNumber)
		}
		if _, err := path.Match(pattern, ""); err != nil {
			return afrIgnore{}, fmt.Errorf("invalid .afrignore pattern on line %d: %w", lineNumber, err)
		}
		ignore.patterns = append(ignore.patterns, ignorePattern{
			value:  pattern,
			isGlob: strings.ContainsAny(pattern, "*?["),
		})
	}
	if err := scanner.Err(); err != nil {
		return afrIgnore{}, fmt.Errorf("read .afrignore: %w", err)
	}
	return ignore, nil
}

func (ignore afrIgnore) matches(relative string) bool {
	relative = strings.TrimPrefix(filepath.ToSlash(relative), "./")
	for _, pattern := range ignore.patterns {
		if pattern.isGlob {
			matched, _ := path.Match(pattern.value, relative)
			if matched {
				return true
			}
			continue
		}
		if relative == pattern.value || strings.HasPrefix(relative, pattern.value+"/") {
			return true
		}
	}
	return false
}
