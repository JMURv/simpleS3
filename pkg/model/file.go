package model

type FileRes struct {
	Path    string `json:"path"`
	ModTime int64  `json:"modTime"`
}
