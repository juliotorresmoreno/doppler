package handler

type ResponseError struct {
	Code    string
	Message string
}

var StatusUnauthorizedMessage = "Unauthorized"
var StatusBadRequestMessage = "Bad Request"
var StatusInternalServerErrorMessage = "Internal Server Error"
