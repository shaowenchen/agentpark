package datastore

import "errors"

var (
	ErrNotFound     = errors.New("not found")
	ErrShareRevoked = errors.New("share revoked")
	ErrShareExpired = errors.New("share expired")
)
