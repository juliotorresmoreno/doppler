package handler

import (
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/juliotorresmoreno/doppler/db"
	"github.com/juliotorresmoreno/doppler/helper"
	"github.com/juliotorresmoreno/doppler/model"
)

type AuthHandler struct{}

func AttachAuth(g *gin.RouterGroup) {
	authHandler := &AuthHandler{}
	g.POST("/sign-in", authHandler.SignIn)
	g.POST("/sign-up", authHandler.SignUp)
}

type SignUpEntity struct {
	Name     string `json:"name"     valid:"alpha,ascii,required"`
	Lastname string `json:"lastname" valid:"alpha,ascii,required"`
	Email    string `json:"email"    valid:"email,ascii,required"`
	Password string `json:"password" valid:"ascii,required"`
}

func (el AuthHandler) SignUp(ctx *gin.Context) {
	payload := new(SignUpEntity)
	err := ctx.Bind(payload)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Code: StatusBadRequestMessage,
		})
		return
	}
	_, err = govalidator.ValidateStruct(payload)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Code:    StatusBadRequestMessage,
			Message: err.Error(),
		})
		return
	}
	valid, _, _, _, _ := helper.VerifyPassword(payload.Password)
	if !valid {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Code:    StatusBadRequestMessage,
			Message: "Password is invalid",
		})
		return
	}
	conn, err := db.GetConnection()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, &ResponseError{
			Code: StatusInternalServerErrorMessage,
		})
		return
	}
	user := new(model.User)
	user.Name = payload.Name
	user.Lastname = payload.Lastname
	user.Email = payload.Email
	user.Password, _ = helper.GeneratePassword(payload.Password)

	tx := conn.Save(user)
	if tx.Error != nil {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Code:    StatusBadRequestMessage,
			Message: "email: El correo electronico ya existe",
		})
		return
	}

	ctx.JSON(200, user)
}

type SignInEntity struct {
	Email    string `json:"email"    valid:"email,ascii,required"`
	Password string `json:"password" valid:"ascii,required"`
}

func (el AuthHandler) SignIn(ctx *gin.Context) {
	payload := new(SignInEntity)
	err := ctx.Bind(payload)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Code: StatusBadRequestMessage,
		})
		return
	}
	_, err = govalidator.ValidateStruct(payload)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Code:    StatusBadRequestMessage,
			Message: err.Error(),
		})
		return
	}
	valid, _, _, _, _ := helper.VerifyPassword(payload.Password)
	if !valid {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Code:    StatusBadRequestMessage,
			Message: "Password is invalid",
		})
		return
	}
	conn, err := db.GetConnection()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, &ResponseError{
			Code: StatusInternalServerErrorMessage,
		})
		return
	}
	user := new(model.User)
	user.Email = payload.Email

	tx := conn.Find(user)
	if tx.Error != nil {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Code:    StatusBadRequestMessage,
			Message: "email: El correo electronico ya existe",
		})
		return
	}

	if helper.ValidatePassword(user.Password, payload.Password) != nil {
		ctx.JSON(http.StatusUnauthorized, &ResponseError{
			Code:    StatusUnauthorizedMessage,
			Message: "El Usuario y/o la contrasena no son validos",
		})
		return
	}

	ctx.JSON(200, user)
}
