package common

import "errors"

// StorageOS API client errors.
var (
	ErrResourceNotFound = errors.New("resource not found")
	ErrUnauthorized     = errors.New("unauthorized")
)
