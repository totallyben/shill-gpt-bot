package api

// ErrorResponse - error response struct
type ErrorResponse struct {
	Errors []string `json:"errors"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ShillLinkReponse struct {
	Link string `json:"link"`
}
