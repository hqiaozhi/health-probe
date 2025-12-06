package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Resp struct {
}

func New() *Resp {
	return &Resp{}
}

func (r *Resp) RESP(ctx *gin.Context, code int, data interface{}, message string) {
	ctx.JSON(http.StatusOK, gin.H{"code": code, "data": data, "message": message})
}

func (r *Resp) RESP_DATA(ctx *gin.Context, data interface{}) {
	r.RESP(ctx, 200, data, "success")
}

func (r *Resp) RESP_OK(ctx *gin.Context) {
	r.RESP(ctx, 200, "", "OK")
}
func (r *Resp) RESP_BAD_REQUEST(ctx *gin.Context, message string) {
	r.RESP(ctx, 400, "", message)
}
func (r *Resp) RESP_NOT_FOUND(ctx *gin.Context, message string) {
	r.RESP(ctx, 404, "", message)
}
func (r *Resp) RESP_ERROR(ctx *gin.Context, message string) {
	r.RESP(ctx, 500, "", message)
}

func (r *Resp) RESP_UNAUTHORIZED(ctx *gin.Context, message string) {
	r.RESP(ctx, 401, "", message)
}
func (r *Resp) RESP_FORBIDDEN(ctx *gin.Context, message string) {
	r.RESP(ctx, 403, "", message)
}
func (r *Resp) RESP_NOT_ACCEPTABLE(ctx *gin.Context, message string) {
	r.RESP(ctx, 406, "", message)
}
func (r *Resp) RESP_CONFLICT(ctx *gin.Context, message string) {
	r.RESP(ctx, 409, "", message)
}
