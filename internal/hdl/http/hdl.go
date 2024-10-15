package http

import (
	"context"
	"fmt"
	"github.com/JMURv/media-server/pkg/config"
	u "github.com/JMURv/media-server/pkg/utils"
	utils "github.com/JMURv/media-server/pkg/utils/http"
	"github.com/JMURv/media-server/pkg/utils/slugify"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Handler struct {
	port     string
	server   *http.Server
	savePath string
	config   *config.HTTPConfig
}

func New(port string, savePath string, config *config.HTTPConfig) *Handler {
	return &Handler{
		port:     port,
		savePath: savePath,
		config:   config,
	}
}

func (h *Handler) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/list/", h.listFiles)
	mux.HandleFunc("/upload", h.createFile)
	mux.HandleFunc("/delete", h.deleteFile)
	mux.HandleFunc("/stream/uploads/", h.stream)
	mux.Handle("/uploads/", http.StripPrefix("/uploads", http.FileServer(http.Dir(h.savePath))))

	h.server = &http.Server{
		Addr:    h.port,
		Handler: mux,
	}

	log.Printf("Server is running on port %v\n", h.port)
	if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting server: %s\n", err)
	}
}

func (h *Handler) Shutdown(ctx context.Context) error {
	if err := h.server.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/stream/uploads/"):]
	path := filepath.Join(h.savePath, name)

	file, err := os.Open(path)
	if err != nil {
		utils.ErrResponse(w, http.StatusNotFound, ErrRetrievingFile)
		return
	}
	defer file.Close()

	switch filepath.Ext(name) {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	case ".webm":
		w.Header().Set("Content-Type", "video/webm")
	default:
		utils.ErrResponse(w, http.StatusUnsupportedMediaType, ErrUnsupportedMediaType)
	}
	w.Header().Set("Transfer-Encoding", "chunked")

	log.Println("Streaming mediafile: ", name)
	buffer := make([]byte, h.config.MaxStreamBuffer)
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			utils.ErrResponse(w, http.StatusInternalServerError, ErrInternal)
			return
		}
		if n == 0 {
			break
		}

		if _, err := w.Write(buffer[:n]); err != nil {
			log.Println("Error writing chunk:", err)
			return
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	page, size := utils.ParsePaginationParams(
		r, h.config.DefaultPage,
		h.config.DefaultSize,
	)

	paths, err := u.ListFilesRecursive(
		filepath.Join(h.savePath, strings.TrimPrefix(r.URL.Path, "/list/")),
	)
	if err != nil {
		log.Println("Error reading directory: ", err)
		utils.ErrResponse(w, http.StatusInternalServerError, ErrReadingDir)
		return
	}

	count := len(paths)
	start := (page - 1) * size
	if start > count {
		start = count
	}

	end := start + size
	if end > count {
		end = count
	}

	totalPages := (count + size - 1) / size
	utils.SuccessPaginatedResponse(
		w, http.StatusOK, utils.PaginatedResponse{
			Data:        paths[start:end],
			Count:       count,
			TotalPages:  totalPages,
			CurrentPage: page,
			HasNextPage: page < totalPages,
		},
	)
}

func (h *Handler) createFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrResponse(w, http.StatusMethodNotAllowed, ErrParsingForm)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.config.MaxUploadSize)
	if err := r.ParseMultipartForm(h.config.MaxUploadSize); err != nil {
		utils.ErrResponse(w, http.StatusBadRequest, ErrFileTooBig)
		return
	}

	path := h.savePath
	if reqPath := r.FormValue("path"); reqPath != "" {
		if !u.IsValidPath(reqPath) {
			utils.ErrResponse(w, http.StatusBadRequest, ErrInvalidPath)
			return
		}
		path = filepath.Join(h.savePath, strings.Trim(reqPath, " /\\"))
	}

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		utils.ErrResponse(w, http.StatusBadRequest, ErrCreatingDir)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		utils.ErrResponse(w, http.StatusBadRequest, ErrRetrievingFile)
		return
	}
	defer file.Close()

	dstPath := filepath.Join(path, slugify.SlugifyFile(handler.Filename))
	if _, err := os.Stat(dstPath); err == nil {
		utils.ErrResponse(w, http.StatusConflict, ErrAlreadyExists)
		return
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		utils.ErrResponse(w, http.StatusInternalServerError, ErrInternal)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		utils.ErrResponse(w, http.StatusInternalServerError, ErrInternal)
		return
	}

	fileURL := fmt.Sprintf("/%s", dstPath)
	log.Printf("File saved: %s\n", fileURL)
	utils.SuccessResponse(w, http.StatusCreated, fileURL)
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.ErrResponse(w, http.StatusMethodNotAllowed, ErrInvalidReqMethod)
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		utils.ErrResponse(w, http.StatusBadRequest, ErrPathNotProvided)
		return
	}

	if err := os.Remove(path); err != nil && os.IsNotExist(err) {
		utils.ErrResponse(w, http.StatusNotFound, err)
		return
	} else if err != nil {
		utils.ErrResponse(w, http.StatusInternalServerError, err)
		return
	}

	log.Printf("File %s deleted successfully\n", path)
	utils.SuccessResponse(w, http.StatusNoContent, "OK")
}
