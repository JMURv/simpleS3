[![Go Coverage](https://github.com/JMURv/simpleS3/wiki/coverage.svg)](https://raw.githack.com/wiki/JMURv/simpleS3/coverage.html)

# simpleS3

`simpleS3` is a lightweight HTTP server that mimics basic S3-like functionalities for file management. It allows users to upload files, list existing files, delete files, and stream media files.

## Features
- Upload files with configurable size limits
- List files with pagination
- Delete files
- Stream media files (e.g., images, videos)

## Configuration

### Example Configuration

You can customize the server's behavior using a YAML configuration file. Below is an example configuration (`example.config.yaml`):

```yaml
port: 8080 # The port on which the server listens for incoming requests
savePath: "uploads" # The directory where uploaded files will be stored

http:
  maxStreamBuffer: 32768 # 32KB chunks | The maximum buffer size for streaming media files
  maxUploadSize: 10485760 # 10 MB | The maximum allowed size for file uploads
  defaultPage: 1 # The default page number for paginated responses
  defaultSize: 40 # The default size number for paginated responses
```

## Endpoint Documentation for simpleS3

This document provides detailed information about the API endpoints available in the simpleS3 server.

### 1. Upload File

#### `POST /upload`

Uploads a file to the server.

#### Request

- **Headers:**
    - `Content-Type`: `multipart/form-data`

- **Form Data:**
    - `file`: (required) The file to upload.
    - `path`: (optional) The directory path where the file will be stored, relative to the `savePath`.

#### Responses

- **201 Created**: File uploaded successfully.
- **400 Bad Request**:
    - Reason: File too large.
    - Reason: Invalid path.
    - Reason: File retrieval error.
- **409 Conflict**:
    - Reason: File already exists.

---

### 2. List Files

#### `GET /list/`

Lists files stored on the server with pagination.

#### Query Parameters

- `page`: (optional) The page number for pagination. Defaults to `1`.
- `size`: (optional) The number of items per page. Defaults to `40`.

#### Responses

- **200 OK**: Returns a JSON object containing:
    - `data`: List of file paths.
    - `count`: Total number of files.
    - `total_pages`: Total number of pages available.
    - `current_page`: The current page number.
    - `has_next_page`: Boolean indicating if there is a next page.
- **500 Internal Server Error**:
    - Reason: Error reading the directory.

---

### 3. Delete File

#### `DELETE /delete`

Deletes a specified file from the server.

#### Query Parameters

- `path`: (required) The path of the file to delete.

#### Responses

- **204 No Content**: File deleted successfully.
- **400 Bad Request**:
    - Reason: No path provided.
- **404 Not Found**:
    - Reason: File not found.
- **500 Internal Server Error**:
    - Reason: Error deleting the file.

---

### 4. Stream Media File

#### `GET /stream/uploads/`

Streams a media file to the client.

#### Request Path

- **name**: The name of the file to stream, appended after `/stream/uploads/`.

#### Responses

- **200 OK**: Streams the requested media file.
- **404 Not Found**:
    - Reason: File not found.
- **415 Unsupported Media Type**:
    - Reason: Unsupported file type (e.g., not an image or video format).

## Run in docker:
```shell
docker run 
--name simpleS3 
-p 8080:8080 
-v path/to/config.yaml:/app/local.config.yaml 
-v named_volume:/app/uploads
jmurv/simpleS3:latest
```

## Using in docker-compose file:
```yaml
  simpleS3:
    container_name: simpleS3
    image: "jmurv/simpleS3:latest"
    restart: always
    volumes:
      - ./path/to/config.yaml:/app/local.config.yaml
      - named_volume:/app/uploads
    ports:
      - "8080:8080"
```