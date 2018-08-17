package scheduler

import (
	"regexp"
	"strings"
	"github.com/Vientiane/errors"
)

//域名服务

//IP验证
var regexpForIP = regexp.MustCompile(`((?:(?:25[0-5]|2[0-4]\d|[01]?\d?\d)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d?\d))`)

//域名后缀验证
var regexpForDomains = []*regexp.Regexp{
	// *.xx or *.xxx.xx
	regexp.MustCompile(`\.(com|com\.\w{2})$`),
	regexp.MustCompile(`\.(gov|gov\.\w{2})$`),
	regexp.MustCompile(`\.(net|net\.\w{2})$`),
	regexp.MustCompile(`\.(org|org\.\w{2})$`),
	// *.xx
	regexp.MustCompile(`\.me$`),
	regexp.MustCompile(`\.biz$`),
	regexp.MustCompile(`\.info$`),
	regexp.MustCompile(`\.name$`),
	regexp.MustCompile(`\.mobi$`),
	regexp.MustCompile(`\.so$`),
	regexp.MustCompile(`\.asia$`),
	regexp.MustCompile(`\.tel$`),
	regexp.MustCompile(`\.tv$`),
	regexp.MustCompile(`\.cc$`),
	regexp.MustCompile(`\.co$`),
	regexp.MustCompile(`\.\w{2}$`),
}

//获取主机名的主域名
func getPrimaryDomain(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host==""{
		return "",errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
			"empty host")
	}
	if regexpForIP.MatchString(host) {
		return host, nil
	}
	var suffixIndex int
	for _, re := range regexpForDomains {
		pos:= re.FindStringIndex(host)
		if pos!=nil{
			suffixIndex = pos[0]
		}
	}
	if suffixIndex>0 {
		var pIndex int
		firstPart := host[:suffixIndex]
		index := strings.LastIndex(firstPart, ".") //得出来主机后面的点的位置，例如www.中点的位置
		if index<0{
			pIndex = 0
		}else {
			pIndex = index + 1
		}
		return host[pIndex:],nil
	}
	return "",errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
		"unrecognized host")
}
