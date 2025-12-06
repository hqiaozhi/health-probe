package users

import (
	"health-probe/internal/svc"
	"health-probe/internal/utils/jwt"
)

type Users struct {
	svcCtx     *svc.ServiceContext
	jwtService *jwt.JwtService
}

func New(svcCtx *svc.ServiceContext) *Users {
	return &Users{
		svcCtx:     svcCtx,
		jwtService: svcCtx.JWT,
	}
}
