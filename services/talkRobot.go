/*
 * @Author: ybc
 * @Date: 2020-07-23 17:42:15
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-17 14:55:53
 * @Description: 钉钉通知
 */
package services

import (
	"encoding/json"
	"errors"

	"github.com/xuanwolei/goutils"
)

const (
	TALK_SEND_ADDRESS string = "https://oapi.dingtalk.com/robot/send?access_token="
)

type TalkRobot struct {
	Token   string
	Param   map[string]interface{}
	Mobiles []string
}

type TalkResponse struct {
	ErrCode int
	ErrMsg  string
}

func NewTalkRobot(token string) *TalkRobot {
	return &TalkRobot{
		Token: token,
	}
}

func (this *TalkRobot) Text(content string) *TalkRobot {
	this.Param = map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}

	return this
}

func (this *TalkRobot) Markdown(title string, content string) *TalkRobot {
	this.Param = map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  content,
		},
	}

	return this
}

func (this *TalkRobot) AtMobiles(mobiles []string) *TalkRobot {
	this.Mobiles = mobiles
	return this
}

func (this *TalkRobot) Send(isAtAll bool) error {
	this.Param["at"] = map[string]interface{}{
		"atMobiles": this.Mobiles,
		"isAtAll":   isAtAll,
	}
	url := TALK_SEND_ADDRESS + this.Token
	req, _ := goutils.NewHttpRequest(url, "POST", this.parseParam())
	req.Headers["Content-Type"] = "application/json"
	body, err := req.Call()
	if err != nil {
		return err
	}
	var response TalkResponse
	json.Unmarshal(body, &response)
	if response.ErrCode != 0 {
		return errors.New(response.ErrMsg)
	}

	return nil
}

func (this *TalkRobot) parseParam() string {
	json, _ := json.Marshal(this.Param)
	return string(json)
}
