package main

import "net/http"

const (
	ErrMessageWrongUser        = "Incorrect username or user does not exist."
	ErrMessageNoPermission     = "User does not have permission for this organization."
	ErrMessageServer           = "Something went wrong. Please try again."
	ErrMessageWrongRequest     = "The request format or parameters are incorrect."
	ErrMessageMethodNotAllowed = "Method not allowed"
)

func sendHttpErr(w http.ResponseWriter, status int) {
	switch status {
	case http.StatusBadRequest:
		http.Error(w, ErrMessageWrongRequest, status)
	case http.StatusUnauthorized:
		http.Error(w, ErrMessageWrongUser, status)
	case http.StatusForbidden:
		http.Error(w, ErrMessageNoPermission, status)
	case http.StatusMethodNotAllowed:
		http.Error(w, ErrMessageMethodNotAllowed, status)
	default:
		http.Error(w, ErrMessageServer, status)
	}

}
