package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/Vientiane/scheduler"
	"strings"
	"github.com/Vientiane/programs/finder/internal"
	"time"
	"github.com/Vientiane/programs/finder/monitor"
	"net/http"
)

//一个简单的爬去图片的爬虫
var(
	firstUrl string
	domains string
	depth uint
	dirPath string
)

func init(){
	flag.StringVar(&firstUrl,"first",
		"https://www.piaohua.com",
		"The first Url which you want to access")
	flag.StringVar(&domains,"domains",
		"piaohua.com,piaowu99.com",
			"please using comma-separated multiple domains")
	flag.UintVar(&depth,"depth",6,"the depth for crawling")
	flag.StringVar(&dirPath,"dir","./pictures",
		"The path which you want to save the image files")
}

func Usage(){
	fmt.Fprintf(os.Stderr,"Usage of %s:\n",os.Args[0])
	fmt.Fprintf(os.Stderr,"\tfinder [flags] \n")
	fmt.Fprintf(os.Stderr,"Flags:\n")
	flag.PrintDefaults()
}

func main() {

	flag.Usage = Usage //使用 finder --help 的时候显示的信息
	flag.Parse()       //将命令行参数解析到变量上面，否则init中的参数均为默认值

	sched := scheduler.New()
	//准备调度器的初始化参数
	domainParts := strings.Split(domains, ",")
	acceptedDomains := []string{}
	for _, domain := range domainParts {
		domain = strings.TrimSpace(domain)
		if domain != "" {
			acceptedDomains = append(acceptedDomains, domain)
		}
	}
	requestArgs := scheduler.RequestArgs{
		AcceptedDomains: acceptedDomains,
		MaxDepth:        uint32(depth),
	}

	dataArgs := scheduler.DataArgs{
		ReqBufferCap:         50,
		ReqMaxBufferNumber:   1000,
		RespBufferCap:        50,
		RespMaxBufferNumber:  10,
		ItemBufferCap:        50,
		ItemMaxBufferNumber:  100,
		ErrorBufferCap:       50,
		ErrorMaxBufferNumber: 1,
	}

	downloaders, err := internal.GetDownloaders(1)
	if err != nil {
		fmt.Printf("An error occurs when creating downloaders: %s", err)
		os.Exit(1)
	}
	analyzers, err := internal.GetAnalyzers(1)
	if err != nil {
		fmt.Printf("An error occurs when creating analyzers: %s", err)
		os.Exit(1)
	}
	pipelines, err := internal.GetPipelines(1, dirPath)
	if err != nil {
		fmt.Printf("An error occurs when creating pipelines: %s", err)
		os.Exit(1)
	}
	moduleArgs := scheduler.ModuleArgs{
		Downloaders: downloaders,
		Analyzers:   analyzers,
		Pipelines:   pipelines,
	}
	err = sched.Init(requestArgs, dataArgs, moduleArgs)
	if err != nil {
		fmt.Printf("An error occurs when initializing scheduler: %s", err)
		os.Exit(1)
	}

	//准备监控参数
	checkInerval := 10* time.Second
	summarizeInterval := 1000 * time.Millisecond
	maxIdleCount := uint(10)
	//开始监控
	checkCountChan := monitor.Monitor(sched, checkInerval, summarizeInterval, maxIdleCount, true)
	//准备调度器的启动参数
	firstHttpReq, err := http.NewRequest("GET", firstUrl, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//开启调度器
	err = sched.Start(firstHttpReq)
	if err != nil {
		fmt.Printf("An error occurs when starting scheduler: %s", err)
	}
	//等待监控结束
	a:= <-checkCountChan
	fmt.Print(a)

}