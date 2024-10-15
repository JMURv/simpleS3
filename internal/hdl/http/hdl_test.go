package http

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/JMURv/media-server/pkg/config"
	utils "github.com/JMURv/media-server/pkg/utils/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const port = ":8080"
const testDir = "./test_uploads"

const createEndpoint = "/upload"
const listEndpoint = "/list"

func setupTestHandler() *Handler {
	return New(
		port,
		testDir,
		&config.HTTPConfig{
			MaxUploadSize:   10 * 1024 * 1024, // 10 MB
			MaxStreamBuffer: 1024,
			DefaultPage:     1,
			DefaultSize:     10,
		},
	)
}

func setupTestDir() {
	if err := os.MkdirAll(testDir, os.ModePerm); err != nil {
		log.Println("Error creating test directory: ", err)
	}
}

func teardownTestDir() {
	if err := os.RemoveAll(testDir); err != nil {
		log.Println("Error removing test directory: ", err)
	}
}

func TestStartAndShutdown(t *testing.T) {
	setupTestDir()
	defer teardownTestDir()
	hdl := New(
		":8083",
		testDir,
		&config.HTTPConfig{
			MaxUploadSize:   10 * 1024 * 1024,
			MaxStreamBuffer: 1024,
			DefaultPage:     1,
			DefaultSize:     10,
		},
	)

	go func() {
		hdl.Start()
	}()
	time.Sleep(100 * time.Millisecond)

	t.Run(
		"Server Running", func(t *testing.T) {
			resp, err := http.Get("http://localhost" + hdl.port + "/list")
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		},
	)

	t.Run(
		"Server Shutdown", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := hdl.Shutdown(ctx)
			assert.NoError(t, err)

			_, err = http.Get("http://localhost" + hdl.port + "/list/")
			assert.Error(t, err)
		},
	)
}

func TestCreateFile(t *testing.T) {
	setupTestDir()
	defer teardownTestDir()
	hdl := setupTestHandler()

	t.Run(
		"Success", func(t *testing.T) {
			fileName := "testfile.txt"
			path := filepath.Join(testDir, fileName)

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			file, _ := writer.CreateFormFile("file", fileName)
			file.Write([]byte("This is a test file."))
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, createEndpoint, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rec := httptest.NewRecorder()
			hdl.createFile(rec, req)

			assert.Equal(t, http.StatusCreated, rec.Result().StatusCode)

			res, _ := io.ReadAll(rec.Result().Body)
			assert.Contains(t, string(res), "test_uploads")
			assert.Contains(t, string(res), fileName)

			_, err := os.Stat(path)
			assert.NoError(t, err)

			err = os.Remove(path)
			assert.NoError(t, err)
		},
	)

	t.Run(
		"Method not allowed", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, createEndpoint, nil)
			rec := httptest.NewRecorder()
			hdl.createFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
		},
	)

	t.Run(
		"Retrieving file error", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, createEndpoint, nil)
			rec := httptest.NewRecorder()
			hdl.createFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		},
	)

	t.Run(
		"File already exists", func(t *testing.T) {
			fileName := "testfile.txt"
			path := filepath.Join(testDir, fileName)

			c, err := os.Create(path)
			assert.Nil(t, err)

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			file, _ := writer.CreateFormFile("file", fileName)
			file.Write([]byte("This is a test file."))
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, createEndpoint, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rec := httptest.NewRecorder()
			hdl.createFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusConflict, res.StatusCode)

			c.Close()
			if err := os.Remove(path); err != nil {
				t.Log(err)
				assert.Nil(t, err)
			}
		},
	)

	t.Run(
		"File too large", func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			file, _ := writer.CreateFormFile("file", "largefile.txt")
			file.Write(bytes.Repeat([]byte("A"), int(hdl.config.MaxUploadSize)+1))
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, createEndpoint, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rec := httptest.NewRecorder()
			hdl.createFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		},
	)

	t.Run(
		"Missing file field", func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			writer.WriteField("path", "some/path")
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, createEndpoint, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rec := httptest.NewRecorder()
			hdl.createFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		},
	)

	t.Run(
		"Invalid path", func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			file, _ := writer.CreateFormFile("file", "testfile.txt")
			file.Write([]byte("This is a test file."))
			writer.WriteField("path", "*123")
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, createEndpoint, body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rec := httptest.NewRecorder()
			hdl.createFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		},
	)

}

func TestListFiles(t *testing.T) {
	setupTestDir()
	defer teardownTestDir()
	hdl := setupTestHandler()

	t.Run(
		"Success", func(t *testing.T) {
			filename1 := "list.txt"
			filename2 := "list1.txt"
			path1 := filepath.Join(testDir, filename1)
			path2 := filepath.Join(testDir, filename2)

			file, err := os.Create(path1)
			assert.Nil(t, err)
			file.Close()

			file, err = os.Create(path2)
			assert.Nil(t, err)
			file.Close()

			req := httptest.NewRequest(http.MethodGet, listEndpoint, nil)
			rec := httptest.NewRecorder()

			hdl.listFiles(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			assert.Contains(t, string(body), filename1)
			assert.Contains(t, string(body), filename2)

			err = os.Remove(path1)
			assert.Nil(t, err)
			err = os.Remove(path2)
			assert.Nil(t, err)
		},
	)

	t.Run(
		"Invalid Path", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/list?path=invalid_path", nil)
			rec := httptest.NewRecorder()

			hdl.listFiles(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			assert.Contains(t, string(body), ErrReadingDir.Error())
		},
	)

	t.Run(
		"Empty Directory", func(t *testing.T) {
			emptyDir := filepath.Join(testDir, "empty")
			err := os.Mkdir(emptyDir, os.ModePerm)
			assert.Nil(t, err)
			defer os.Remove(emptyDir)

			req := httptest.NewRequest(http.MethodGet, "/list?path=empty", nil)
			rec := httptest.NewRecorder()

			hdl.listFiles(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			assert.NotContains(t, string(body), ErrReadingDir.Error())
			assert.Contains(t, string(body), `"data":[]`)
		},
	)

	t.Run(
		"Pagination", func(t *testing.T) {
			filename1 := "list1.txt"
			filename2 := "list2.txt"
			filename3 := "list3.txt"
			path1 := filepath.Join(testDir, filename1)
			path2 := filepath.Join(testDir, filename2)
			path3 := filepath.Join(testDir, filename3)

			f, err := os.Create(path1)
			assert.Nil(t, err)
			defer f.Close()

			f1, err := os.Create(path2)
			assert.Nil(t, err)
			defer f1.Close()

			f2, err := os.Create(path3)
			assert.Nil(t, err)
			defer f2.Close()

			defer os.Remove(path1)
			defer os.Remove(path2)
			defer os.Remove(path3)

			// Test first page of pagination
			req := httptest.NewRequest(http.MethodGet, "/list?page=1&size=2", nil)
			rec := httptest.NewRecorder()

			hdl.listFiles(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			assert.Contains(t, string(body), filename1)
			assert.Contains(t, string(body), filename2)
			assert.NotContains(t, string(body), filename3)

			// Test second page of pagination
			req = httptest.NewRequest(http.MethodGet, "/list?page=2&size=2", nil)
			rec = httptest.NewRecorder()

			hdl.listFiles(rec, req)

			res = rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)

			body, _ = io.ReadAll(res.Body)
			assert.Contains(t, string(body), filename3)
		},
	)

	t.Run(
		"Out of Range Page", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/list?page=999&size=10", nil)
			rec := httptest.NewRecorder()

			hdl.listFiles(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)

			body, _ := io.ReadAll(res.Body)
			assert.Contains(t, string(body), `"data":[]`)
		},
	)

	t.Run(
		"Invalid Query Parameters", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/list?page=invalid&size=invalid", nil)
			rec := httptest.NewRecorder()

			hdl.listFiles(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)

			resp := &utils.PaginatedResponse{}

			err := json.NewDecoder(res.Body).Decode(resp)
			require.Nil(t, err)

			assert.Equal(t, 1, resp.CurrentPage)
			assert.Equal(t, false, resp.HasNextPage)
		},
	)
}

func TestDeleteFile(t *testing.T) {
	setupTestDir()
	defer teardownTestDir()

	hdl := setupTestHandler()

	t.Run(
		"Success", func(t *testing.T) {
			file, err := os.Create("./test_uploads/delete.txt")
			assert.Nil(t, err)
			file.Close()

			req := httptest.NewRequest(http.MethodDelete, "/delete?path=test_uploads/delete.txt", nil)
			rec := httptest.NewRecorder()

			hdl.deleteFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusNoContent, res.StatusCode)

			_, err = os.Stat("./test_uploads/delete.txt")
			assert.True(t, os.IsNotExist(err))
		},
	)

	t.Run(
		"Method not allowed", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/delete?path=test_uploads/delete.txt", nil)
			rec := httptest.NewRecorder()

			hdl.deleteFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
		},
	)

	t.Run(
		"Filename not provided", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/delete", nil)
			rec := httptest.NewRecorder()

			hdl.deleteFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		},
	)

	t.Run(
		"File not found", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/delete?path=test_uploads/nonexistent.txt", nil)
			rec := httptest.NewRecorder()

			hdl.deleteFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusNotFound, res.StatusCode)
		},
	)

}

func TestStream(t *testing.T) {
	setupTestDir()
	defer teardownTestDir()

	handler := setupTestHandler()

	t.Run(
		"Success MP4 Streaming", func(t *testing.T) {
			path := filepath.Join(testDir, "testfile.mp4")
			expType := "video/mp4"
			expText := "This is a test video file."

			err := os.WriteFile(path, []byte(expText), 0644)
			assert.Nil(t, err)
			defer os.Remove(path)

			req := httptest.NewRequest(http.MethodGet, "/stream/uploads/testfile.mp4", nil)
			rec := httptest.NewRecorder()

			handler.stream(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, expType, res.Header.Get("Content-Type"))

			body, err := io.ReadAll(res.Body)
			assert.Nil(t, err)
			assert.Equal(t, expText, string(body))
		},
	)

	t.Run(
		"Success WEBM Streaming", func(t *testing.T) {
			path := filepath.Join(testDir, "testfile.webm")
			expType := "video/webm"
			expText := "This is a test video file."

			err := os.WriteFile(path, []byte(expText), 0644)
			assert.Nil(t, err)
			defer os.Remove(path)

			req := httptest.NewRequest(http.MethodGet, "/stream/uploads/testfile.webm", nil)
			rec := httptest.NewRecorder()

			handler.stream(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, expType, res.Header.Get("Content-Type"))

			body, err := io.ReadAll(res.Body)
			assert.Nil(t, err)
			assert.Equal(t, expText, string(body))
		},
	)

	t.Run(
		"Success JPEG Streaming", func(t *testing.T) {
			path := filepath.Join(testDir, "testfile.jpeg")
			expType := "image/jpeg"
			expText := "This is a test video file."

			err := os.WriteFile(path, []byte(expText), 0644)
			assert.Nil(t, err)
			defer os.Remove(path)

			req := httptest.NewRequest(http.MethodGet, "/stream/uploads/testfile.jpeg", nil)
			rec := httptest.NewRecorder()

			handler.stream(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, expType, res.Header.Get("Content-Type"))

			body, err := io.ReadAll(res.Body)
			assert.Nil(t, err)
			assert.Equal(t, expText, string(body))
		},
	)

	t.Run(
		"Success PNG Streaming", func(t *testing.T) {
			path := filepath.Join(testDir, "testfile.png")
			expType := "image/png"
			expText := "This is a test video file."

			err := os.WriteFile(path, []byte(expText), 0644)
			assert.Nil(t, err)
			defer os.Remove(path)

			req := httptest.NewRequest(http.MethodGet, "/stream/uploads/testfile.png", nil)
			rec := httptest.NewRecorder()

			handler.stream(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, expType, res.Header.Get("Content-Type"))

			body, err := io.ReadAll(res.Body)
			assert.Nil(t, err)
			assert.Equal(t, expText, string(body))
		},
	)

	t.Run(
		"Success GIF Streaming", func(t *testing.T) {
			path := filepath.Join(testDir, "testfile.gif")
			expType := "image/gif"
			expText := "This is a test video file."

			err := os.WriteFile(path, []byte(expText), 0644)
			assert.Nil(t, err)
			defer os.Remove(path)

			req := httptest.NewRequest(http.MethodGet, "/stream/uploads/testfile.gif", nil)
			rec := httptest.NewRecorder()

			handler.stream(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, expType, res.Header.Get("Content-Type"))

			body, err := io.ReadAll(res.Body)
			assert.Nil(t, err)
			assert.Equal(t, expText, string(body))
		},
	)

	t.Run(
		"Unsupported Media Type", func(t *testing.T) {
			path := filepath.Join(testDir, "testfile.txt")
			err := os.WriteFile(path, []byte("This is a test file."), 0644)
			assert.Nil(t, err)
			defer os.Remove(path)

			req := httptest.NewRequest(http.MethodGet, "/stream/uploads/testfile.txt", nil)
			rec := httptest.NewRecorder()

			handler.stream(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusUnsupportedMediaType, res.StatusCode)
		},
	)

	t.Run(
		"File Not Found", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/stream/uploads/nonexistent.mp4", nil)
			rec := httptest.NewRecorder()

			handler.stream(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusNotFound, res.StatusCode)
		},
	)
}
