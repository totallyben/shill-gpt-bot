package api

import "errors"

var (
	ErrUnknownError     = errors.New("an unknown error occurred")
	ErrRequestBindError = errors.New("unable to bind request object")

	ErrWalletNotFound = errors.New("wallet not found")

	ErrReplyPersonaNotFound = errors.New("persona not found")

	ErrTrollNotFound               = errors.New("troll not found")
	ErrTrollUnableToConfirmTwitter = errors.New("unable to confirm troll on twitter")

	ErrTwitterEmptyTweet          = errors.New("no tweet was received to generate a reply to")
	ErrTwitterTweetAlreadyTrolled = errors.New("you have already trolled this tweet")
	ErrTwitterTweetIsTroll        = errors.New("you can't troll a troll")
	ErrTwitterReplyTextNoMatch    = errors.New("the reply on twitter does not match with troll")

	ErrOpenAiReplyLength = errors.New("we had a problem generating a reply with the correct character cound, please try again")

	ErrMockFatalError    = errors.New("a fatal error occurred")
	ErrMockNotFound      = errors.New("not found")
	ErrMockNotAuthorised = errors.New("not authorised")

	ErrTransactionsSearch = errors.New("could not search transactions")

	ErrPresaleNotFound               = errors.New("presale not found")
	ErrPresaleUnableToUpdateProgress = errors.New("unable to update presale progress")
)
