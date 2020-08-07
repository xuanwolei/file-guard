/*
 * @Author: ybc
 * @Date: 2020-06-29 19:30:45
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-07 14:54:14
 * @Description: file content
 */

package services

import (
	"sync"

	"time"

	"flag"
	"github.com/hpcloud/tail"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

var AppConfig *ini.File

var NoticeChan = make(chan *NoticeContent)
var Exit = make(chan int)
var Wait sync.WaitGroup

type Guard struct {
	Section   *ini.Section
	Config    *Config
	Files     []*FileInfo
	MatchFunc FilterFunc
	Tails     []*tail.Tail
}

type Config struct {
	LogFile        string
	LogDriver      string
	MatchPreg      string
	FilterPreg     string
	NoticeToken    string
	NoticeLevel    string
	LogCheckLength string
}

type NoticeContent struct {
	Path  string
	Line  *tail.Line
	Guard *Guard
}

type FilterFunc func(pattern string, text string) bool

const (
	LOG_DRIVER_ERROR  string = "error"
	LOG_DRIVER_CUSTOM string = "custom"
	DEFAULT_SECTION   string = "DEFAULT"
)

var (
	DEFAULT_CONFIG map[string]string = map[string]string{
		"log_driver":       LOG_DRIVER_ERROR,
		"match_preg":       "(?i)error",
		"filter_preg":      "",
		"notice_level":     "5",
		"log_check_length": "30",
	}
	Guards     []*Guard
	ConfigFile *string = flag.String("c", "./conf/app.ini", "Ini file path")
)

func init() {
	flag.Parse()
	conf, err := LoadConfig(*ConfigFile)
	if err != nil {
		panic(err)
	} 
	AppConfig = conf
}

func Reload() {
	log.Info("guard restart")
	for _, guard := range Guards {
		if len(guard.Tails) < 1 {
			continue
		}
		for _, tail := range guard.Tails {
			tail.Stop()
		}
	}
	Guards = Guards[0:0]

	LoadSections()
}

func LoadConfig(configFile string) (*ini.File, error) {
	conf, err := ini.Load(configFile)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func LoadSections() {
	defaultSection := AppConfig.Section("DEFAULT")
	defaultConfig := defaultSection.KeysHash()
	sections := AppConfig.Sections()
	for _, section := range sections {
		if section.Name() == "DEFAULT" {
			continue
		}
		hashConfig := section.KeysHash()
		config := StringMapSetDefaultVal(hashConfig, defaultConfig)
		guard := &Guard{
			Section: section,
			Config: &Config{
				LogFile:        config["log_file"],
				LogDriver:      config["log_driver"],
				MatchPreg:      config["match_preg"],
				FilterPreg:     config["filter_preg"],
				NoticeToken:    config["notice_token"],
				NoticeLevel:    config["notice_level"],
				LogCheckLength: config["log_check_length"],
			},
			MatchFunc: MatchString,
		}
		Guards = append(Guards, guard)
		go guard.Run()
	}
	return
}

func Listen() {
	LoadSections()
	go HandleNotice()
	go HandelTick()

	<-Exit
}

func HandelTick() {
	t1 := time.Tick(3600 * time.Second)
	for {
		select {
		case <-t1:
			log.Info("tick run")
			Reload()
		}
	}
}

func StringMapSetDefaultVal(hash map[string]string, defaultHash map[string]string) map[string]string {
	for k, v := range defaultHash {
		if hash[k] != "" {
			continue
		} else if v == "" {
			hash[k] = DEFAULT_CONFIG[k]
		}
		hash[k] = v
	}

	return hash
}

func (this *Guard) Run() {
	file, err := PathExists(this.Config.LogFile)
	if err != nil {
		log.Error("path:", this.Config.LogFile, err.Error())
		return
	}

	var files []*FileInfo
	if file == nil {
		var readFile = make(chan *FileInfo)
		FindFiles(this.Config.LogFile, readFile, true)
		log.Debug("start:", this.Config.LogFile)
		for f := range readFile {
			log.Info("file:", f.Path)
			files = append(files, f)
		}
		log.Debug("finish")
	} else {
		files = append(files, &FileInfo{
			File: file,
			Path: this.Config.LogFile,
		})
	}

	this.Files = files
	this.listen()
}

func (this *Guard) listen() {
	for _, f := range this.Files {
		go this.tail(f.Path)
	}
}

func (this *Guard) tail(path string) {
	config := tail.Config{
		ReOpen:    true,                                 // 重新打开
		Follow:    true,                                 // 是否跟随
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // 从文件的哪个地方开始读
		MustExist: false,                                // 文件不存在不报错
		Poll:      true,
	}
	t, err := tail.TailFile(path, config)
	this.Tails = append(this.Tails, t)
	if err != nil {
		log.Error(err.Error())
		return
	}
	for line := range t.Lines {
		this.handle(path, line)
	}
	return
}

func (this *Guard) handle(path string, line *tail.Line) {
	if !this.MatchFunc(this.Config.MatchPreg, line.Text) {
		log.Debug("未匹配", line.Text)
		return
	}
	if this.Config.FilterPreg != "" && this.MatchFunc(this.Config.FilterPreg, line.Text) {
		log.Debug("已过滤", line.Text)
		return
	}
	//send notice
	NoticeChan <- &NoticeContent{
		Line:  line,
		Guard: this,
		Path:  path,
	}
	return
}
