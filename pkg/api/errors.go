package api

import "errors"

var (
	ErrUnknownError     = errors.New("an unknown error occurred")
	ErrRequestBindError = errors.New("unable to bind request object")

	ErrOpenAiReplyLength = errors.New("we had a problem generating a reply with the correct character count, please try again")

	ErrMockFatalError    = errors.New("a fatal error occurred")
	ErrMockNotFound      = errors.New("not found")
	ErrMockNotAuthorised = errors.New("not authorised")

	ErrShillNotFound = errors.New("could not find that shill request")
)
