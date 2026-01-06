package sdk

import (
	"github.com/gin-gonic/gin"
	"gucooing/lolo/db"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/protocol/quick"
	"strconv"
)

func getLoginResultV2(user *db.OFQuick) *quick.LoginResultV2 {
	result := &quick.LoginResultV2{
		ExtInfo:       new(quick.ExtInfo),
		IsAdult:       true,
		UAge:          9999,
		CkPlayTime:    0,
		GuestRealName: 1,
		Id:            0,
		Message:       "",
		AuthToken:     user.AuthToken,
		UserData:      getUserDataV2(user),
		CheckRealname: 0,
	}
	return result
}

func getUserDataV2(user *db.OFQuick) *quick.UserDataV2 {
	data := &quick.UserDataV2{
		Uid:       strconv.FormatUint(uint64(user.ID), 10),
		Username:  user.Username,
		Mobile:    "188****8888",
		IsGuest:   "",
		RegDevice: user.RegDevice,
		SexType:   "",
		IsMbUser:  1,
		IsSnsUser: 0,
		Token:     user.UserToken,
	}
	return data
}

func (s *Server) loginByNameV2(c *gin.Context) {
	req := new(quick.LoginByNameRequest)
	rsp := quick.NewResponse()
	defer c.JSON(200, rsp)
	if err := alg.DecryptedData(c, &req); err != nil {
		rsp.SetError("解密失败")
		log.App.Debugf("gin req autoLogin error: %v", err)
		return
	}
	user, err := db.OrCreateOFQuick(req.Username, req.Password)
	if err != nil {
		rsp.SetError("解密失败")
		return
	}
	if user.Password != req.Password {
		rsp.SetError("解密失败")
		return
	}

	// 更新token
	if user.AuthToken == "" {
		user.AuthToken = s.GenToken(user.ID ^ authXor)
	}
	user.UserToken = s.GenToken(user.ID)
	if err := db.UpOFQuick(user); err != nil {
		rsp.SetError("更新失败")
		return
	}

	rsp.SetData(getLoginResultV2(user))
}

func (s *Server) autoLoginV2(c *gin.Context) {
	req := new(quick.AutoLoginRequest)
	rsp := quick.NewResponse()
	defer c.JSON(200, rsp)
	if err := alg.DecryptedData(c, &req); err != nil {
		rsp.SetError("解密失败")
		log.App.Debugf("gin req autoLogin error: %v", err)
		return
	}
	token, err := s.ToToken(req.AuthToken)
	if err != nil {
		rsp.SetError("解密失败")
		return
	}
	user, err := db.GetOFQuick(token.ID ^ authXor)
	if err != nil {
		rsp.SetError("没有该账号")
		return
	}
	// 更新token
	// user.AuthToken = s.GenToken(user.ID^13745713)
	user.UserToken = s.GenToken(user.ID)
	if err := db.UpOFQuick(user); err != nil {
		rsp.SetError("更新失败")
		return
	}

	rsp.SetData(getLoginResultV2(user))
}
