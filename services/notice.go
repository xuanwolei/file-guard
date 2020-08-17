/*
 * @Author: ybc
 * @Date: 2020-07-23 16:46:50
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-17 14:56:02
 * @Description: 通知
 */

package services

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strconv"
)

var (
	noticeLevel map[string]*NoticeRule = map[string]*NoticeRule{
		"1": &NoticeRule{
			IntervalTime: 10,
			LimitTime:    3600,
			LimitNum:     360,
		},
		"2": &NoticeRule{
			IntervalTime: 60,
			LimitTime:    3600,
			LimitNum:     60,
		},
		"3": &NoticeRule{
			IntervalTime: 600,
			LimitTime:    3600,
			LimitNum:     5,
		},
		"4": &NoticeRule{
			IntervalTime: 1800,
			LimitTime:    86400,
			LimitNum:     40,
		},
		"5": &NoticeRule{
			IntervalTime: 3600,
			LimitTime:    86400,
			LimitNum:     24,
		},
		"6": &NoticeRule{
			IntervalTime: 7200,
			LimitTime:    86400,
			LimitNum:     10,
		},
		"7": &NoticeRule{
			IntervalTime: 7200 * 2,
			LimitTime:    86400,
			LimitNum:     5,
		},
		"8": &NoticeRule{
			IntervalTime: 86400,
			LimitTime:    86400,
			LimitNum:     1,
		},
	}
	statistic map[string]int
	table     *XwTable
	localIp   string
)

type NoticeRule struct {
	IntervalTime int64 //每次通知间隔,单位秒
	LimitTime    int64 //通知限制时间,单位秒
	LimitNum     int64 //限制时间内最多通知次数
}

func init() {
	table = NewXwTable()
	localIp, _ = GetLocalIp()
	return
}

func HandleNotice() {
	for {
		select {
		case notice := <-NoticeChan:
			fmt.Println("receive:", notice)
			notice.run()
		}
	}
}

func (this *NoticeContent) run() {
	if err := this.check(); err != nil {
		log.Info("notice:" + err.Error())
		return
	}
	this.report()
}

func (this *NoticeContent) check() error {
	var (
		limitNumKey string = this.parseKey("ln", true)
		IntervalKey string = this.parseKey("inter", true)
	)
	rule := noticeLevel[this.Guard.Config.NoticeLevel]
	if table.GetInt(limitNumKey) > rule.LimitNum {
		return errors.New("noticeMaxLimit:" + this.Line.Text)
	}
	if table.GetInt(IntervalKey) > 0 {
		return errors.New("noticeInterval:" + this.Line.Text)
	}

	table.Incrby(limitNumKey, 1)
	table.Expire(limitNumKey, rule.LimitTime)
	if rule.IntervalTime > 0 {
		table.SetExInt(IntervalKey, rule.IntervalTime, 1)
	}

	return nil
}

func (this *NoticeContent) report() {
	instance := NewTalkRobot(this.Guard.Config.NoticeToken)
	title := "项目：" + this.Guard.Section.Name()
	content := "- 项目:" + this.Guard.Section.Name() + "\n"
	content += "- IP :" + localIp + "\n"
	content += "- 文件:" + this.Path + "\n"
	content += "- 时间：" + this.Line.Time.Format("2006-01-02 15:04:05") + "\n"
	content += "## 内容:\n```\n" + this.Line.Text + "\n"

	var atMobiles []string
	if this.Guard.Config.NoticeMobile != "" {
		atMobiles = append(atMobiles, this.Guard.Config.NoticeMobile)
		content += "叮叮叮：@" + this.Guard.Config.NoticeMobile + "\n"
	}
	content += "```"
	log.Info("atMobile", atMobiles)
	if err := instance.Markdown(title, content).AtMobiles(atMobiles).Send(false); err != nil {
		log.Error("notice fail:", err.Error(), "title:", title, ",content:", content)
	}
	log.Info("notice:", title)
	return
}

func (this *NoticeContent) parseKey(val string, isConnetText bool) string {
	text := ""
	length := len(this.Line.Text)
	checkLength, _ := strconv.Atoi(this.Guard.Config.LogCheckLength)
	skipLength, _ := strconv.Atoi(this.Guard.Config.LogSkipLength)
	if length > checkLength {
		length = checkLength
	}
	if length < skipLength {
		skipLength = 0
	}
	if isConnetText {
		text = this.Line.Text[skipLength:length]
	}
	return this.Guard.Section.Name() + val + text
}
