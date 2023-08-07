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
	data := new(model.User)
	data.Name = payload.Name
	data.Lastname = payload.Lastname
	data.Email = payload.Email
	data.Password, _ = helper.GeneratePassword(payload.Password)

	tx := conn.Save(data)
	if tx.Error != nil {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Code:    StatusBadRequestMessage,
			Message: "email: El correo electronico ya existe",
		})
		return
	}

	token := helper.GenerateToken(data)

	ctx.JSON(200, Session{
		Token: token,
		User: &User{
			Id:       data.Id,
			Name:     data.Name,
			Lastname: data.Lastname,
			Email:    data.Email,
		},
	})
}

type User struct {
	Id       uint   `json:"id"`
	Name     string `json:"name"`
	Lastname string `json:"lastname"`
	Email    string `json:"email"`
}

type Session struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
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
		ctx.JSON(http.StatusUnauthorized, &ResponseError{
			Code:    StatusUnauthorizedMessage,
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
	token := helper.GenerateToken(user)

	ctx.JSON(200, Session{
		Token: token,
		User: &User{
			Id:       user.Id,
			Name:     user.Name,
			Lastname: user.Lastname,
			Email:    user.Email,
		},
	})
}
