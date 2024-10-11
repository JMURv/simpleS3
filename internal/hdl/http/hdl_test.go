package http

import (
	"bytes"
	"github.com/JMURv/media-server/pkg/config"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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

			req := httptest.NewRequest(http.MethodDelete, "/delete?filename=delete.txt", nil)
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
			req := httptest.NewRequest(http.MethodGet, "/delete?filename=delete.txt", nil)
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
			req := httptest.NewRequest(http.MethodDelete, "/delete?filename=nonexistent.txt", nil)
			rec := httptest.NewRecorder()

			hdl.deleteFile(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusNotFound, res.StatusCode)
		},
	)

	//t.Run(
	//	"Error deleting file", func(t *testing.T) {
	//		file, err := os.Create("./test_uploads/protected.txt")
	//		assert.Nil(t, err)
	//		file.Close()
	//
	//		err = os.Chmod("./test_uploads/protected.txt", 0444)
	//		assert.Nil(t, err)
	//
	//		req := httptest.NewRequest(http.MethodDelete, "/delete?filename=protected.txt", nil)
	//		rec := httptest.NewRecorder()
	//
	//		hdl.deleteFile(rec, req)
	//
	//		res := rec.Result()
	//		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	//
	//		err = os.Chmod("./test_uploads/protected.txt", 0644)
	//		assert.Nil(t, err)
	//		err = os.Remove("./test_uploads/protected.txt")
	//		assert.Nil(t, err)
	//	},
	//)
}

func TestStream(t *testing.T) {
	setupTestDir()
	defer teardownTestDir()

	handler := setupTestHandler()

	t.Run(
		"Success", func(t *testing.T) {
			path := filepath.Join(testDir, "testfile.mp4")
			expType := "video/mp4"
			expText := "This is a test video file."

			err := os.WriteFile(path, []byte(expText), 0644)
			assert.Nil(t, err)

			req := httptest.NewRequest(http.MethodGet, "/stream/uploads/testfile.mp4", nil)
			rec := httptest.NewRecorder()

			handler.stream(rec, req)

			res := rec.Result()
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, expType, res.Header.Get("Content-Type"))

			body, _ := io.ReadAll(res.Body)
			assert.Equal(t, expText, string(body))

			err = os.Remove(path)
			assert.Nil(t, err)
		},
	)
}
