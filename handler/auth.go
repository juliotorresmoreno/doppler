package handler

import (
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct{}

func AttachAuth(g *gin.RouterGroup) {
	authHandler := &AuthHandler{}
	g.GET("/sign-up", authHandler.SignUp)
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
			Message: "Bad Request",
		})
	}
	_, err = govalidator.ValidateStruct(payload)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, &ResponseError{
			Message: "Bad Request",
		})
	}
	ctx.JSON(200, "ok")
}
