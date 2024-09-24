package helpers

import (
	"fmt"
	"os"
	"path/filepath"
)

func ListFilesInDir(dir string) (map[string]struct{}, error) {
	files := make(map[string]struct{})
	err := filepath.Walk(
		dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relPath, err := filepath.Rel(dir, path)
				if err != nil {
					return err
				}
				files["/uploads/"+relPath] = struct{}{}
			}
			return nil
		},
	)

	if err != nil {
		return nil, err
	}
	return files, nil
}

func DeleteUnreferencedFiles(uploadDir string, localPaths, pathsFromDB map[string]struct{}) error {
	for file := range localPaths {
		if _, ok := pathsFromDB[file]; !ok {
			fullPath := filepath.Join(uploadDir, file[len("/uploads/"):])
			fmt.Printf("Deleting unreferenced file: %s\n", fullPath)
			if err := os.Remove(fullPath); err != nil {
				return err
			}
		}
	}
	return nil
}
