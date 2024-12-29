package utils

import (
	"github.com/JMURv/simple-s3/pkg/model"
	"os"
	"path/filepath"
	"strings"
)

func IsValidPath(p string) bool {
	return !strings.ContainsAny(p, `<>:"|?*`)
}

func SearchBySubStr(p []model.FileRes, subStr string) []model.FileRes {
	res := make([]model.FileRes, 0, len(p)/2)

	for _, v := range p {
		if strings.Contains(strings.ToLower(v.Path), strings.ToLower(subStr)) {
			res = append(res, v)
		}
	}
	return res
}

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
				if err = collectFiles(filepath.Join(path, entry.Name())); err != nil {
					return err
				}
			} else {
				modTime := int64(0)
				if fileData, err := entry.Info(); err == nil {
					modTime = fileData.ModTime().Unix()
				}

				paths = append(
					paths, model.FileRes{
						Path:    filepath.Join("/", path, entry.Name()),
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
