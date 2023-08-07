package handler

import (
	"fmt"
	"reflect"
)

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

func UpdateModel(model interface{}, v interface{}) {
	value := reflect.Indirect(reflect.ValueOf(v))
	nfields := value.NumField()

	m := reflect.Indirect(reflect.ValueOf(model))

	for i := 0; i < nfields; i++ {
		key := value.Type().Field(i).Name
		value := value.Field(i)

		dest := m.FieldByName(key)
		if !dest.CanInterface() {
			continue
		}

		tmp := value.Interface()
		if dest.CanInt() && value.CanInt() && value.Int() > 0 {
			dest.SetInt(value.Int())
		} else if dest.CanFloat() && value.CanFloat() && value.Float() > 0 {
			dest.SetFloat(value.Float())
		} else if dest.CanUint() && value.CanUint() && value.Uint() > 0 {
			dest.SetUint(value.Uint())
		} else if fmt.Sprintf("%T", tmp) == "string" {
			dest.SetString(value.String())
		} else if dest.CanInterface() && value.CanInterface() && value.Interface() != nil {
			dest.Set(value)
		}
	}
}
