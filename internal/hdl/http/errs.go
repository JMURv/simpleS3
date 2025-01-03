package http

import "errors"

var ErrFileTooBig = errors.New("file too big")
var ErrAlreadyExists = errors.New("file already exists")
var ErrInvalidReqMethod = errors.New("invalid request method")
var ErrInternal = errors.New("internal error")

var ErrMissingQuery = errors.New("missing query")
var ErrInvalidPath = errors.New("invalid path")
var ErrCreatingDir = errors.New("error creating directory")
var ErrPathNotProvided = errors.New("path not provided")
var ErrRetrievingFile = errors.New("error retrieving file")
var ErrParsingForm = errors.New("error parsing form")
var ErrReadingDir = errors.New("error reading directory")
var ErrUnsupportedMediaType = errors.New("unsupported media type")
