/*
 * @Author: ybc
 * @Date: 2020-07-23 16:46:50
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-25 16:38:30
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
		return errors.New("noticeMaxLimit:" + this.Event.Text)
	}
	if table.GetInt(IntervalKey) > 0 {
		return errors.New("noticeInterval:" + this.Event.Text)
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
	var atMobiles []string
	if this.Guard.Config.NoticeMobile != "" {
		atMobiles = append(atMobiles, this.Guard.Config.NoticeMobile)
	}
	title, content := this.buildMarkdown()
	log.Info("atMobile", atMobiles)
	if err := instance.Markdown(title, content).AtMobiles(atMobiles).Send(false); err != nil {
		log.Error("notice fail:", err.Error(), "title:", title, ",content:", content)
	}
	log.Info("notice:", title)
	return
}

// buildMarkdown 在发送前按钉钉单条消息上限裁剪，保证代码块完整闭合。
func (this *NoticeContent) buildMarkdown() (string, string) {
	maxBytes := this.Guard.Config.NoticeMaxBytes
	if maxBytes <= 0 {
		maxBytes = 12000
	}
	reservedBytes := this.Guard.Config.NoticeReservedBytes
	if reservedBytes <= 0 {
		reservedBytes = 1024
	}
	title := "项目：" + truncateUTF8(this.Guard.Section.Name(), 128)
	prefix := "- 项目:" + truncateUTF8(this.Guard.Section.Name(), 128) + "\n"
	prefix += "- IP :" + truncateUTF8(localIp, 64) + "\n"
	prefix += "- 文件:" + truncateUTF8(this.Path, 256) + "\n"
	prefix += "- 时间：" + this.Event.Time.Format("2006-01-02 15:04:05") + "\n"
	prefix += "## 内容:\n```\n"
	suffix := "\n```"
	budget := maxBytes - len(prefix) - len(suffix)
	reservedBudget := maxBytes - reservedBytes
	if reservedBudget < budget {
		budget = reservedBudget
	}
	if budget < 0 {
		budget = 0
	}
	body := this.Event.Text
	if len(body) > budget {
		body = truncateWithSuffix(body, budget, "\n[通知内容已按钉钉长度限制截断]")
	}
	content := prefix + body + suffix
	// 元数据异常长时仍确保总内容不超限；优先保留正文和 Markdown 闭合。
	if len(content) > maxBytes {
		content = truncateWithSuffix(prefix+body, maxBytes-len(suffix), "\n[通知内容已截断]") + suffix
	}
	return title, content
}

func (this *NoticeContent) parseKey(val string, isConnetText bool) string {
	text := ""
	length := len(this.Event.Text)
	checkLength, _ := strconv.Atoi(this.Guard.Config.LogCheckLength)
	skipLength, _ := strconv.Atoi(this.Guard.Config.LogSkipLength)
	checkLength += skipLength
	if length > checkLength {
		length = checkLength
	}
	if length < skipLength {
		skipLength = 0
	}
	log.Info("skipLength:", skipLength, ",length:", length)
	if isConnetText {
		text = this.Event.Text[skipLength:length]
	}
	return this.Guard.Section.Name() + val + text
}
