package utils

import (
	"testing"
)

func TestGenerateUniqueFilename(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		allUserFilenames []string
		expected         string
	}{
		{
			name:             "No existing files",
			filename:         "resume.pdf",
			allUserFilenames: []string{"other.pdf", "something.txt"},
			expected:         "resume.pdf",
		},
		{
			name:             "Exact match exists",
			filename:         "resume.pdf",
			allUserFilenames: []string{"resume.pdf"},
			expected:         "resume (1).pdf",
		},
		{
			name:             "Add suffix to conflict",
			filename:         "resume.pdf",
			allUserFilenames: []string{"resume.pdf", "resume (1).pdf", "resume (2).pdf"},
			expected:         "resume (3).pdf",
		},
		{
			name:             "Input already numbered but base is used",
			filename:         "resume (1).pdf",
			allUserFilenames: []string{"resume.pdf", "resume (1).pdf"},
			expected:         "resume (1) (1).pdf",
		},
		{
			name:             "Input numbered, base not used",
			filename:         "resume (1).pdf",
			allUserFilenames: []string{"resume (1).pdf", "resume (1) (1).pdf"},
			expected:         "resume (1) (2).pdf",
		},
		{
			name:             "Base file and deep numbered exist",
			filename:         "resume.pdf",
			allUserFilenames: []string{"resume (1).pdf", "resume (1) (1).pdf"},
			expected:         "resume.pdf",
		},
		{
			name:             "Filename with special characters",
			filename:         "file@#$%.txt",
			allUserFilenames: []string{"file@#$%.txt"},
			expected:         "file@#$% (1).txt",
		},
		{
			name:             "Filename with multiple dots",
			filename:         "archive.tar.gz",
			allUserFilenames: []string{"archive.tar.gz"},
			expected:         "archive.tar (1).gz",
		},
		{
			name:             "Dot file no conflict",
			filename:         ".gitignore",
			allUserFilenames: []string{},
			expected:         ".gitignore",
		},
		{
			name:             "Dot file conflict",
			filename:         ".gitignore",
			allUserFilenames: []string{".gitignore"},
			expected:         " (1).gitignore",
		},
		{
			name:             "No extension",
			filename:         "README",
			allUserFilenames: []string{"README", "README (1)"},
			expected:         "README (2)",
		},
		{
			name:             "Empty filename",
			filename:         "",
			allUserFilenames: []string{""},
			expected:         " (1)",
		},
		{
			name:             "Case sensitive names do not conflict",
			filename:         "Doc.txt",
			allUserFilenames: []string{"doc.txt"},
			expected:         "Doc.txt",
		},
		{
			name:             "High numbered input with base and lower exist",
			filename:         "data (100).txt",
			allUserFilenames: []string{"data.txt", "data (1).txt", "data (99).txt"},
			expected:         "data (100).txt",
		},
		{
			name:             "Base exists but input is high numbered",
			filename:         "data (10).txt",
			allUserFilenames: []string{"data.txt", "data (1).txt"},
			expected:         "data (10).txt",
		},
		{
			name:             "Base is missing but numbered exist",
			filename:         "data.txt",
			allUserFilenames: []string{"data (2).txt", "data (3).txt"},
			expected:         "data.txt",
		},
		{
			name:             "Complex nested suffixes",
			filename:         "resume (1) (1).pdf",
			allUserFilenames: []string{"resume.pdf", "resume (1).pdf", "resume (1) (1).pdf"},
			expected:         "resume (1) (1) (1).pdf",
		},
		{
			name:             "Special letters not considered suffix",
			filename:         "data (abc).txt",
			allUserFilenames: []string{"data.txt"},
			expected:         "data (abc).txt",
		},
		{
			name:             "Spaces and numbering",
			filename:         "my file.pdf",
			allUserFilenames: []string{"my file.pdf", "my file (1).pdf"},
			expected:         "my file (2).pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateUniqueFilename(tt.filename, tt.allUserFilenames)
			if got != tt.expected {
				t.Errorf("FAILED [%s]: got %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}
