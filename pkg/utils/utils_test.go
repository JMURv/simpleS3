package utils

import (
	"github.com/JMURv/simple-s3/pkg/model"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestIsValidPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"validpath", true},
		{"invalid:path", false},
		{"invalid|path", false},
		{"invalid<path", false},
		{"invalid>path", false},
		{"invalid*path", false},
		{"invalid?path", false},
		{"invalid\"path", false},
		{"valid_path_123", true},
	}

	for _, tt := range tests {
		t.Run(
			tt.path, func(t *testing.T) {
				result := IsValidPath(tt.path)
				assert.Equal(t, tt.expected, result)
			},
		)
	}
}

func TestSearchBySubStr(t *testing.T) {
	files := []model.FileRes{
		{Path: "/path/to/file1.txt"},
		{Path: "/path/to/anotherfile.doc"},
		{Path: "/different/path/file2.txt"},
		{Path: "/some/other/path/file3.pdf"},
	}

	tests := []struct {
		subStr   string
		expected []model.FileRes
	}{
		{"file", files[:4]},
		{"another", files[1:2]},
		{"different", files[2:3]},
		{"notfound", []model.FileRes{}},
	}

	for _, tt := range tests {
		t.Run(
			tt.subStr, func(t *testing.T) {
				result := SearchBySubStr(files, tt.subStr)
				assert.Equal(t, tt.expected, result)
			},
		)
	}
}

func TestListFilesRecursive(t *testing.T) {
	tempDir := t.TempDir()

	files := []struct {
		path     string
		contents string
	}{
		{"file1.txt", "content1"},
		{"dir1/file2.txt", "content2"},
		{"dir2/file3.txt", "content3"},
	}

	for _, file := range files {
		filePath := filepath.Join(tempDir, file.path)
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			t.Fatalf("Failed to create directories: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(file.contents), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	expectedPaths := []model.FileRes{
		{Path: filepath.Join("/", tempDir, "file1.txt")},
		{Path: filepath.Join("/", tempDir, "dir1", "file2.txt")},
		{Path: filepath.Join("/", tempDir, "dir2", "file3.txt")},
	}

	result, err := ListFilesRecursive(tempDir)
	assert.NoError(t, err)
	assert.Len(t, result, len(expectedPaths))

	for _, expPath := range expectedPaths {
		found := false
		for _, resPath := range result {
			if resPath.Path == expPath.Path {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected path not found: %v", expPath.Path)
	}
}
