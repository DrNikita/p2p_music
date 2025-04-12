package db

import (
	"errors"
)

var (
	errDuplicateKey = errors.New("song already exists")
	errSongNotFound = errors.New("song not found")
)

var (
	errTestArraysNotEqual = errors.New("arrays are not equal")
)
