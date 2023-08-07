package handler

import (
	"net/http"
	"strconv"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/db"
	"github.com/juliotorresmoreno/doppler/helper"
	"github.com/juliotorresmoreno/doppler/model"
)

type Server struct {
}

func AttachServer(g *gin.RouterGroup) {
	server := &Server{}
	g.GET("/:id", server.Get)
	g.GET("", server.FindAll)
	g.POST("", server.Post)
	g.PATCH("/:id", server.Patch)
	g.DELETE("/:id", server.Delete)
}

func (svr *Server) Get(ctx *gin.Context) {
	tmpSession, ok := ctx.Keys["session"]
	if !ok {
		ctx.JSON(http.StatusUnauthorized, &ResponseError{
			Code:    StatusUnauthorizedMessage,
			Message: StatusUnauthorizedMessage,
		})
		return
	}
	session := tmpSession.(*Session)

	vid := ctx.Param("id")
	id, _ := strconv.Atoi(vid)
	result, err := svr.get(uint(id), session)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, &ResponseError{
			Code:    StatusInternalServerErrorMessage,
			Message: err.Error(),
		})
		return
	}
	ctx.JSON(200, result)
}

func (svr *Server) FindAll(ctx *gin.Context) {
	tmpSession, ok := ctx.Keys["session"]
	if !ok {
		ctx.JSON(http.StatusUnauthorized, &ResponseError{
			Code:    StatusUnauthorizedMessage,
			Message: StatusUnauthorizedMessage,
		})
		return
	}
	session := tmpSession.(*Session)

	conf, _ := config.GetConfig()
	limit := helper.ParseIntParams(ctx.Query("limit"), conf.Limit)
	offset := helper.ParseIntParams(ctx.Query("offset"), 0)

	result, total, err := svr.getAll(session, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, &ResponseError{
			Code:    StatusInternalServerErrorMessage,
			Message: err.Error(),
		})
		return
	}
	ctx.JSON(200, &ResponseData{
		Data:   result,
		Limit:  limit,
		Offset: offset,
		Total:  total,
	})
}

type ServerEntity struct {
	Name        string `json:"name"        valid:"alphanum,ascii,required"`
	Description string `json:"description" valid:"alphanum,ascii"`
	IpAddress   string `json:"ip_address"  valid:"ip,ascii,required"`
}

func (svr *Server) Post(ctx *gin.Context) {
	tmpSession, ok := ctx.Keys["session"]
	if !ok {
		ctx.JSON(http.StatusUnauthorized, &ResponseError{
			Code:    StatusUnauthorizedMessage,
			Message: StatusUnauthorizedMessage,
		})
		return
	}
	session := tmpSession.(*Session)

	payload := new(ServerEntity)
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

	data := &model.Server{
		Name:        payload.Name,
		Description: payload.Description,
		IpAddress:   payload.IpAddress,
		OwnerID:     session.User.Id,
	}
	conn, err := db.GetConnection()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, &ResponseError{
			Code: StatusInternalServerErrorMessage,
		})
		return
	}

	tx := conn.Save(data)
	if tx.Error != nil {
		ctx.JSON(http.StatusInternalServerError, &ResponseError{
			Code:    StatusInternalServerErrorMessage,
			Message: tx.Error.Error(),
		})
		return
	}

	result, err := svr.get(data.Id, session)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, &ResponseError{
			Code:    StatusInternalServerErrorMessage,
			Message: err.Error(),
		})
		return
	}
	ctx.JSON(200, result)
}

func (svr *Server) get(id uint, session *Session) (*model.Server, error) {
	result := new(model.Server)
	conn, err := db.GetConnection()
	if err != nil {
		return result, err
	}

	tx := conn.Preload("Owner", "id = ?", session.User.Id).First(result, id)
	if tx.Error != nil {
		return result, tx.Error
	}

	return result, nil
}

func (svr *Server) getAll(session *Session, limit, offset int) ([]*model.Server, int64, error) {
	result := make([]*model.Server, 0)
	conn, err := db.GetConnection()
	if err != nil {
		return result, 0, err
	}

	tx := conn.Preload("Owner", "id = ?", session.User.Id).
		Where("owner_id = ?", session.User.Id).
		Limit(limit).
		Offset(offset).
		Find(&result)
	if tx.Error != nil {
		return result, 0, tx.Error
	}

	var total int64 = 0
	tx = conn.Table("servers").
		Where("owner_id = ?", session.User.Id).
		Count(&total)
	if tx.Error != nil {
		return result, 0, tx.Error
	}

	return result, total, nil
}

func (svr *Server) Patch(ctx *gin.Context) {

}

func (svr *Server) Delete(ctx *gin.Context) {

}
