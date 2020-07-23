/*
 * @Author: ybc
 * @Date: 2020-07-23 16:46:50
 * @LastEditors: ybc
 * @LastEditTime: 2020-07-23 20:20:21
 * @Description: file content
 */

package services

import (
	"fmt"
	log "github.com/sirupsen/logrus"
)

var (
	noticeLevel map[string]*NoticeRule = map[string]*NoticeRule{
		"1": &NoticeRule{
			IntervalTime:      60,
			LimitTime:         3600,
			LimitNum:          30,
			ErrorIntervalTime: 3600,
		},
	}
	statistic map[string]int
)

type NoticeRule struct {
	IntervalTime      int //每次通知间隔,单位秒
	LimitTime         int //通知限制时间,单位秒
	LimitNum          int //限制时间内最多通知次数
	ErrorIntervalTime int //相同错误间隔时间
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
	this.check()
	this.report()
}

func (this *NoticeContent) check() {
	// rule := noticeLevel[this.Guard.Config.NoticeLevel]
	this.Guard.Section.Name()
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
