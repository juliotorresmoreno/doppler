package handler

type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var StatusUnauthorizedMessage = "Unauthorized"
var StatusBadRequestMessage = "Bad Request"
var StatusInternalServerErrorMessage = "Internal Server Error"
