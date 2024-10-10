package handlers

import "errors"

var ErrFileTooBig = errors.New("file too big")
var ErrFilenameNotProvided = errors.New("filename not provided")
var ErrInvalidReqMethod = errors.New("invalid request method")
var ErrInternal = errors.New("internal error")
var ErrRetrievingFile = errors.New("error retrieving file")
var ErrParsingForm = errors.New("error parsing form")
var ErrAlreadyExists = errors.New("file already exists")

var ErrReadingDir = errors.New("error reading directory")
