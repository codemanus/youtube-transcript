package videoid

import "errors"

var (
	ErrEmpty      = errors.New("url or video id is required")
	ErrInvalidURL = errors.New("not a recognized youtube url or video id")
)
