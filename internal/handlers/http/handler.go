package http

import (
	"context"
	"fmt"
	"github.com/JMURv/media-server/pkg/consts"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Handler struct {
	port   string
	server *http.Server
}

func New(port string) *Handler {
	return &Handler{
		port: port,
	}
}

func (h *Handler) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", h.uploadFile)
	mux.HandleFunc("/stream/uploads/", h.streamImage)
	mux.Handle("/uploads/", http.StripPrefix("/uploads", http.FileServer(http.Dir(consts.SavePath))))

	h.server = &http.Server{
		Addr:    h.port,
		Handler: mux,
	}

	fmt.Printf("Server is running on port %v\n", h.port)
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

func (h *Handler) streamImage(w http.ResponseWriter, r *http.Request) {
	imageName := r.URL.Path[len("/stream/uploads/"):]
	filePath := path.Join(consts.SavePath, imageName)

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Transfer-Encoding", "chunked")

	buffer := make([]byte, 1024*32) // 32KB chunks
	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
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

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request) {
	log.Printf("Saving new file...")
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := fmt.Sprintf(
		"%s%d%s",
		strings.Split(filepath.Base(handler.Filename), ".")[0],
		time.Now().Unix(),
		filepath.Ext(handler.Filename),
	)
	dst, err := os.Create(filepath.Join(consts.SavePath, filename))
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}

	fileURL := fmt.Sprintf("/uploads/%s", filename)
	fmt.Printf("File saved: %s\n", fileURL)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"url": "%s"}`, fileURL)
}
