package utils

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var suffixRegex = regexp.MustCompile(`^(.*)\s\((\d+)\)$`)

func GenerateUniqueFilename(filename string, allUserFilenames []string) string {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	// Determine if the exact input exists in the list
	exactMatch := false
	for _, f := range allUserFilenames {
		if f == filename {
			exactMatch = true
			break
		}
	}

	// If not conflicted, return as is
	if !exactMatch {
		return filename
	}

	// When there’s a conflict, decide on base to append suffix to
	// If base has a suffix like (1), DO NOT strip it; treat it as part of the base
	normalizedBase := base

	usedSuffixes := make(map[int]bool)
	for _, existing := range allUserFilenames {
		if filepath.Ext(existing) != ext {
			continue
		}
		existingBase := strings.TrimSuffix(existing, ext)

		// Match pattern "(n)" at the end
		if match := suffixRegex.FindStringSubmatch(existingBase); match != nil {
			if match[1] == normalizedBase {
				if num, err := strconv.Atoi(match[2]); err == nil {
					usedSuffixes[num] = true
				}
			}
		} else if existingBase == normalizedBase {
			// base name already used
			usedSuffixes[0] = true
		}
	}

	// Find next available suffix
	suffixes := make([]int, 0, len(usedSuffixes))
	for k := range usedSuffixes {
		suffixes = append(suffixes, k)
	}
	sort.Ints(suffixes)

	nextSuffix := 1
	for _, s := range suffixes {
		if s == nextSuffix {
			nextSuffix++
		}
	}

	return fmt.Sprintf("%s (%d)%s", normalizedBase, nextSuffix, ext)
}
