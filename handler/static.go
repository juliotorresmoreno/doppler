package handler

import (
	"os"
	"path"

	"github.com/gin-gonic/gin"
)

const staticPath = "./website/dist"

func AttachStatic(r *gin.Engine) {
	r.NoRoute(func(ctx *gin.Context) {
		if ctx.Request.Method != "GET" {
			ctx.File(staticPath)
			return
		}
		fpath := path.Join(staticPath, ctx.Request.URL.Path)
		_, err := os.Stat(fpath)
		if err != nil {
			ctx.File(staticPath)
			return
		}
		ctx.File(fpath)
	})
}
