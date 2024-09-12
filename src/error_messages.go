package main

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
)

type ErrorInfo struct {
	status int
	reason string
}

func sendHttpErr(w http.ResponseWriter, error_info ErrorInfo) {

	json_response := fmt.Sprintf(`{"reason": "%s"}`, error_info.reason)
	http.Error(w, json_response, error_info.status)

	return

}
