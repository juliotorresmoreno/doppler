package handler

type ResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResponseData struct {
	Data   interface{} `json:"data"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Total  int64       `json:"total"`
}

var StatusUnauthorizedMessage = "Unauthorized"
var StatusBadRequestMessage = "Bad Request"
var StatusInternalServerErrorMessage = "Internal Server Error"
