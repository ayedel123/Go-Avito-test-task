package errinfo

import (
	"fmt"
	"net/http"
)

const (
	ErrMessageWrongUser        = "Incorrect username or user does not exist."
	ErrMessageNoPermission     = "User does not have permission."
	ErrMessageServer           = "Something went wrong. Please try again."
	ErrMessageWrongRequest     = "The request format or parameters are incorrect."
	ErrMessageMethodNotAllowed = "Method not allowed"
	ErrMessageTenderNotFound   = "Tender not Found"
	ErrMessageBidNotFound      = "Bid not Found"
)

type ErrorInfo struct {
	Status int
	Reason string
}

func SendHttpErr(w http.ResponseWriter, errorInfo ErrorInfo) {

	jsonResponse := fmt.Sprintf(`{"reason": "%s"}`, errorInfo.Reason)

	http.Error(w, jsonResponse, errorInfo.Status)

}

func (err *ErrorInfo) Init(status int, reason string) {
	err.Status = status
	err.Reason = reason
}
