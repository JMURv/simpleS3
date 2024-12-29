package http

import (
	"context"
	_ "github.com/JMURv/simple-s3/docs"
	"github.com/JMURv/simple-s3/pkg/config"
	"github.com/JMURv/simple-s3/pkg/model"
	u "github.com/JMURv/simple-s3/pkg/utils"
	utils "github.com/JMURv/simple-s3/pkg/utils/http"
	"github.com/JMURv/simple-s3/pkg/utils/slugify"
	swag "github.com/swaggo/http-swagger"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func (h *Handler) Start(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/swagger/", swag.WrapHandler)

	mux.HandleFunc("/list", h.listFiles)
	mux.HandleFunc("/search", h.searchFiles)
	mux.HandleFunc("/upload", h.createFile)
	mux.HandleFunc("/delete", h.deleteFile)
	mux.HandleFunc("/stream/uploads/", h.stream)
	mux.Handle("/uploads/", http.StripPrefix("/uploads", http.FileServer(http.Dir(h.savePath))))

	h.server = &http.Server{
		Addr:    h.port,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		if err := h.server.Shutdown(ctx); err != nil {
			log.Fatalf("Error shutting down server: %s\n", err)
		}
	}()

	log.Printf("Server is running on port %v\n", h.port)
	if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting server: %s\n", err)
	}
}

// stream streams a media file based on the given path
// @Summary Stream a media file
// @Description Streams a media file for the given path
// @Param path path string true "File path"
// @Produce  media/*
// @Success 200
// @Failure 404 {object} utils.ErrorResponse
// @Failure 415 {object} utils.ErrorResponse
// @Router /stream/uploads/{path} [get]
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

// searchFiles search files by name in a directory with pagination
// @Summary Search files
// @Description Retrieve a list of files matching the given name from a directory with pagination
// @Param q query string true "Search query"
// @Param path query string false "Directory path"
// @Param page query int false "Page number" default(1)
// @Param size query int false "Number of items per page" default(10)
// @Success 200 {object} utils.PaginatedResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /search [get]
func (h *Handler) searchFiles(w http.ResponseWriter, r *http.Request) {
	q := strings.Trim(r.URL.Query().Get("q"), " /\\")
	if q == "" {
		utils.ErrResponse(w, http.StatusBadRequest, ErrMissingQuery)
		return
	}

	path := strings.Trim(r.URL.Query().Get("path"), " /\\")
	if !u.IsValidPath(path) {
		utils.ErrResponse(w, http.StatusBadRequest, ErrInvalidPath)
		return
	}

	paths, err := u.ListFilesRecursive(
		filepath.Join(h.savePath, path),
	)
	if err != nil {
		log.Println("Error reading directory: ", err)
		utils.ErrResponse(w, http.StatusInternalServerError, ErrReadingDir)
		return
	}

	paths = u.SearchBySubStr(paths, q)
	page, size := utils.ParsePaginationParams(
		r, h.config.DefaultPage,
		h.config.DefaultSize,
	)

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
	utils.SuccessDataResponse(
		w, http.StatusOK, utils.PaginatedResponse{
			Data:        paths[start:end],
			Count:       count,
			TotalPages:  totalPages,
			CurrentPage: page,
			HasNextPage: page < totalPages,
		},
	)
}

// listFiles lists all files in a directory with pagination
// @Summary List files with pagination
// @Description Retrieve a list of files from a directory with pagination
// @Param path query string false "Directory path"
// @Param page query int false "Page number" default(1)
// @Param size query int false "Number of items per page" default(10)
// @Success 200 {object} utils.PaginatedResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /list [get]
func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	page, size := utils.ParsePaginationParams(
		r, h.config.DefaultPage,
		h.config.DefaultSize,
	)

	path := strings.Trim(r.URL.Query().Get("path"), " /\\")
	if !u.IsValidPath(path) {
		utils.ErrResponse(w, http.StatusBadRequest, ErrInvalidPath)
		return
	}

	files, err := u.ListFilesRecursive(
		filepath.Join(h.savePath, path),
	)
	if err != nil {
		log.Println("Error reading directory: ", err)
		utils.ErrResponse(w, http.StatusInternalServerError, ErrReadingDir)
		return
	}

	count := len(files)
	start := (page - 1) * size
	if start > count {
		start = count
	}

	end := start + size
	if end > count {
		end = count
	}

	totalPages := (count + size - 1) / size
	utils.SuccessDataResponse(
		w, http.StatusOK, utils.PaginatedResponse{
			Data:        files[start:end],
			Count:       count,
			TotalPages:  totalPages,
			CurrentPage: page,
			HasNextPage: page < totalPages,
		},
	)
}

// createFile uploads a new file to the server
// @Summary Upload a new file
// @Description Uploads a file to a specified path
// @Accept multipart/form-data
// @Param path formData string false "Directory path"
// @Param file formData file true "File to upload"
// @Success 201 {object} model.FileRes
// @Failure 400 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /upload [post]
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

	dstPath := filepath.Join(path, slugify.Filename(handler.Filename))
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

	utils.SuccessDataResponse(
		w, http.StatusCreated, &model.FileRes{
			Path:    "/" + dstPath,
			ModTime: time.Now().Unix(),
		},
	)
}

// deleteFile deletes a specified file
// @Summary Delete a file
// @Description Deletes a file from the server
// @Param path query string true "File path"
// @Success 204
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /delete [delete]
func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.ErrResponse(w, http.StatusMethodNotAllowed, ErrInvalidReqMethod)
		return
	}

	path := strings.Trim(r.URL.Query().Get("path"), " /\\")
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
