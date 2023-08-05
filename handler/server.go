package handler

import (
	"github.com/gin-gonic/gin"
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

}

func (svr *Server) FindAll(ctx *gin.Context) {

}

func (svr *Server) Post(ctx *gin.Context) {

}

func (svr *Server) Patch(ctx *gin.Context) {

}

func (svr *Server) Delete(ctx *gin.Context) {

}
