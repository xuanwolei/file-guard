/*
 * @Author: ybc
 * @Date: 2020-07-23 16:46:50
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-06 20:10:36
 * @Description: file content
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
)

type NoticeRule struct {
	IntervalTime int64 //每次通知间隔,单位秒
	LimitTime    int64 //通知限制时间,单位秒
	LimitNum     int64 //限制时间内最多通知次数
}

func init() {
	table = NewXwTable()
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
		return errors.New("达到通知上限:" + this.Line.Text)
	}
	if table.GetInt(IntervalKey) > 0 {
		return errors.New("在通知间隔内:" + this.Line.Text)
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
	content += "- 文件:" + this.Path + "\n"
	content += "- 时间：" + this.Line.Time.Format("2006-01-02 15:04:05") + "\n"
	content += "## 内容:\n```\n" + this.Line.Text + "\n```"
	if err := instance.Markdown(title, content).Send(false); err != nil {
		log.Error("notice fail:", err.Error(), "title:", title, ",content:", content)
	}
	return
}

func (this *NoticeContent) parseKey(val string, isConnetText bool) string {
	text := ""
	length := len(this.Line.Text)
	checkLength, _ := strconv.Atoi(this.Guard.Config.LogCheckLength)
	if length > checkLength {
		length = checkLength
	}
	if isConnetText {
		text = this.Line.Text[0:length]
	}
	return this.Guard.Section.Name() + val + text
}
