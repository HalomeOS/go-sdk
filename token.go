package go_sdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type ReqToken struct {
	Account    string `json:"account"`
	ApiKey     string `json:"apiKey"`
	ExpireTime int64  `json:"expireTime"`
}

type RespToken struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// CreateToken 创建token
// account 账号
// apiKey 密钥
// expireTime 过期时间
// gatewayUrl 网关地址
// return token  token
// return err  错误信息
func CreateToken(account string, apiKey string, expireTime int64, gatewayUrl string) (string, error) {
	reqToken := &ReqToken{
		Account:    account,
		ApiKey:     apiKey,
		ExpireTime: expireTime,
	}
	jsonData, err := json.Marshal(reqToken)
	if err != nil {
		return "", err
	}
	// 发送POST请求
	resp, err := http.Post(gatewayUrl+"/u/createToken", "application/json", bytes.NewBuffer(jsonData))
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("request failed Status: " + resp.Status)
	}
	// 读取响应体
	var respToken RespToken
	if err = json.NewDecoder(resp.Body).Decode(&respToken); err != nil {
		return "", err
	}
	if respToken.Code != 200 {
		return "", errors.New("create token failed, err: " + respToken.Message)
	}
	return respToken.Data, nil
}
