package model

type FileListReq struct {
	Path string `json:"path"`
}

type FileRes struct {
	Path    string `json:"path"`
	ModTime int64  `json:"modTime"`
}
