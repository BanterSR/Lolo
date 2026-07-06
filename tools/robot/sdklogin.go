package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/protocol/quick"
)

// singKey 与服务端 pkg/alg 内的 singKey 一致：既是 AES-ECB-128 密钥，又是签名盐。
var singKey = []byte("0b2a18e45d7df321")

var httpClient = &http.Client{Timeout: 15 * time.Second}

// SdkLogin 走完整 SDK HTTP 登录链，返回 (sdkUid, gateToken)：
//
//  1. POST /v1/user/loginByName (V1, 无需签名) -> uid + userToken
//  2. POST /v2/users/checkLogin (V2, 需签名)   -> user_token(即 GateToken)
//
// gateToken 随后用于 TCP VerifyLoginTokenReq.LoginToken。
func SdkLogin(base, username, password string) (string, string, error) {
	// 1) loginByName —— 账号不存在则自动创建
	loginReq := &quick.LoginByNameRequest{
		Username:    username,
		Password:    password,
		Platform:    1,
		ProductCode: "lolo",
		ChannelCode: "lolo",
		ClientLang:  "zh",
		DeviceId:    username,
	}
	var loginResp struct {
		Result  bool                `json:"result"`
		Data    quick.LoginResultV1 `json:"data"`
		Message string              `json:"message"`
	}
	if err := sdkPost(base+"/v1/user/loginByName", loginReq, false, &loginResp); err != nil {
		return "", "", fmt.Errorf("loginByName: %w", err)
	}
	if !loginResp.Result || loginResp.Data.UserData == nil {
		return "", "", fmt.Errorf("loginByName 失败: %s", loginResp.Message)
	}
	uid := loginResp.Data.UserData.Uid
	userToken := loginResp.Data.UserData.Token

	// 2) checkLogin —— 签发 GateToken 并写入 OFQuickCheck
	checkReq := &quick.CheckLoginRequest{
		Uid:         uid,
		UserName:    username,
		Token:       userToken,
		PackageName: "com.of.lolo",
		Platform:    1,
	}
	var checkResp struct {
		Result  bool                     `json:"result"`
		Data    quick.CheckLoginResponse `json:"data"`
		Message string                   `json:"message"`
	}
	if err := sdkPost(base+"/v2/users/checkLogin", checkReq, true, &checkResp); err != nil {
		return "", "", fmt.Errorf("checkLogin: %w", err)
	}
	if !checkResp.Result || checkResp.Data.UserToken == "" {
		return "", "", fmt.Errorf("checkLogin 失败: %s", checkResp.Message)
	}
	return uid, checkResp.Data.UserToken, nil
}

// sdkPost 加密请求体（AES-ECB + base64.RawStd 写入 data 字段），按需附带签名，
// 再解析服务端返回的明文 JSON。签名与服务端用同一函数在同一份明文上计算，保证一致。
func sdkPost(fullURL string, body any, sign bool, out any) error {
	plain, err := json.Marshal(body)
	if err != nil {
		return err
	}
	cipher, err := alg.AESECB128Encode(singKey, plain)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("data", base64.RawStdEncoding.EncodeToString(cipher))
	if sign {
		form.Set("sign", alg.SingBytes(plain, singKey))
	}

	resp, err := httpClient.PostForm(fullURL, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}
