/*
 * @Author: ybc
 * @Date: 2020-06-29 19:30:45
 * @LastEditors: ybc
 * @LastEditTime: 2020-07-21 11:34:59
 * @Description: file content
 */

package services

import (
	"gopkg.in/ini.v1"
)

var AppConfig *ini.File

type Guard struct {
	Section *ini.Section
	Config  *Config
}

type Config struct {
	LogFile     string
	LogDriver   string
	FilterPreg  string
	NoticeToken string
	NoticeLevel string
}

var (
	DEFAULT_CONFIG map[string]string = map[string]string{
		"log_driver":  "laravel",
		"filter_preg": "",
		"NoticeLevel": "1",
	}
)

func init() {
	conf, err := LoadConfig("./conf/app.ini")
	if err != nil {
		panic(err)
	}
	AppConfig = conf
	// s := conf.Section("dev")
	// if s == nil{
	// 	fmt.Println("not nil")
	// }
	// fmt.Println(s.Key("a").String())
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
				LogFile:     config["log_file"],
				LogDriver:   config["log_driver"],
				FilterPreg:  config["filter_preg"],
				NoticeToken: config["notice_token"],
				NoticeLevel: config["notice_level"],
			},
		}
		go guard.Run()
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

}
