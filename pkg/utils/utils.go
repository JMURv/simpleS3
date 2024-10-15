package utils

import (
	"github.com/JMURv/media-server/pkg/model"
	"os"
	"path/filepath"
)

func ListFilesRecursive(path string) ([]model.FileRes, error) {
	paths := make([]model.FileRes, 0, 50)

	var collectFiles func(string) error
	collectFiles = func(path string) error {
		dirs, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		for _, entry := range dirs {
			if entry.IsDir() {
				if err := collectFiles(filepath.Join(path, entry.Name())); err != nil {
					return err
				}
			} else {
				modTime := int64(0)
				if fileData, err := entry.Info(); err == nil {
					modTime = fileData.ModTime().Unix()
				}

				paths = append(
					paths, model.FileRes{
						Path:    filepath.Join(path, entry.Name()),
						ModTime: modTime,
					},
				)
			}
		}
		return nil
	}

	if err := collectFiles(path); err != nil {
		return nil, err
	}
	return paths, nil
}
