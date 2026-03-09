package domain

import "errors"

var (
	ErrNotFound              = errors.New("extension not found")
	ErrAlreadyInstalled      = errors.New("extension already installed")
	ErrNotInstalled          = errors.New("extension not installed")
	ErrVersionNotFound       = errors.New("compatible version not found")
	ErrAllSourcesUnavailable = errors.New("download failed: all sources unavailable")
)
