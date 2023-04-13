package handler

import (
	"github.com/gin-gonic/gin"
)

const staticPath = "./website/dist"

func AttachStatic(g *gin.Engine) {
	g.Static("", staticPath)
	g.NoRoute(func(c *gin.Context) {
		c.File(staticPath)
	})
}
