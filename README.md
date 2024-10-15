[![Go Coverage](https://github.com/JMURv/simpleS3/wiki/coverage.svg)](https://raw.githack.com/wiki/JMURv/simpleS3/coverage.html)

# simpleS3

`simpleS3` is a lightweight HTTP server that mimics basic S3-like functionalities for file management. It allows users to upload files, list existing files, delete files, and stream media files.

## Features
- Upload files with configurable size limits
- List files with pagination
- Delete files
- Stream media files (e.g., images, videos)
- Generated swagger documentation avaliable at: `/swagger/index.html`

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

## Run in docker:
```shell
docker run 
--name simple-s3 
-p 8080:8080 
-v path/to/config.yaml:/app/local.config.yaml 
-v named_volume:/app/uploads
jmurv/simple-s3:latest
```

## Using in docker-compose file:
```yaml
  simple-s3:
    container_name: simple-s3
    image: "jmurv/simple-s3:latest"
    restart: always
    volumes:
      - ./path/to/config.yaml:/app/local.config.yaml
      - named_volume:/app/uploads
    ports:
      - "8080:8080"
```