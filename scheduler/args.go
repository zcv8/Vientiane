package scheduler

import (
	"github.com/Vientiane/module"
	"github.com/Vientiane/errors"
)

//参数容器的接口类型
type Args interface {
	//用于检测数据的有效性
	Check() error
}

//请求相关的参数容器类型（声明爬虫爬取的广度和深度）
type RequestArgs struct {
	//代表爬虫可以接受的域名，不在范围内的域名将会自动忽略
	AcceptedDomains []string `json:"accepted_primary_domains"`
	//代表爬虫爬取的最大深度
	MaxDepth uint32 `json:"max_depth"`
}

func(args *RequestArgs)Check()error {
	if args.AcceptedDomains == nil {
		return errors.NewIllegalParameterError("nil accepted primary domain list")
	}
	return nil
}

//组件相关的参数容器类型
type ModuleArgs struct {
	//下载器列表
	Downloaders []module.Downloader
	//分析器列表
	Analyzers []module.Analyzer
	//条目处理管道列表
	Pipelines []module.Pipeline
}

func(args *ModuleArgs)Check()error {
	if len(args.Downloaders) == 0 {
		return errors.NewIllegalParameterError("empty downloader list")
	}
	if len(args.Analyzers) == 0 {
		return errors.NewIllegalParameterError("empty analyzer list")
	}
	if len(args.Pipelines) == 0 {
		return errors.NewIllegalParameterError("empty pipeline list")
	}
	return nil
}


//数据相关的参数容器类型（调度器通过缓冲池来传递数据）
type DataArgs struct {
	//请求缓冲器的容量
	ReqBufferCap uint32 `json:"req_buffer_cap"`
	//请求缓冲器的最大数量
	ReqMaxBufferNumber uint32 `json:"req_max_buffer_number"`
	//响应缓冲器的容量
	RespBufferCap uint32 `json:"resp_buffer_cap"`
	//响应缓冲器的最大数量
	RespMaxBufferNumber uint32 `json:"resp_max_buffer_number"`
	//条目缓冲器的容量
	ItemBufferCap uint32 `json:"item_buffer_cap"`
	//条目缓冲器的最大数量
	ItemMaxBufferNumber uint32 `json:"item_max_buffer_number"`
	//错误缓冲器的容量
	ErrorBufferCap uint32 `json:"error_buffer_number"`
	//错误缓冲器的最大数量
	ErrorMaxBufferNumber uint32 `json:"error_max_buffer_number"`
}

func (args *DataArgs) Check() error {
	if args.ReqBufferCap == 0 {
		return errors.NewIllegalParameterError("zero request buffer capacity")
	}
	if args.ReqMaxBufferNumber == 0 {
		return errors.NewIllegalParameterError("zero max request buffer number")
	}
	if args.RespBufferCap == 0 {
		return errors.NewIllegalParameterError("zero response buffer capacity")
	}
	if args.RespMaxBufferNumber == 0 {
		return errors.NewIllegalParameterError("zero max response buffer number")
	}
	if args.ItemBufferCap == 0 {
		return errors.NewIllegalParameterError("zero item buffer capacity")
	}
	if args.ItemMaxBufferNumber == 0 {
		return errors.NewIllegalParameterError("zero max item buffer number")
	}
	if args.ErrorBufferCap == 0 {
		return errors.NewIllegalParameterError("zero error buffer capacity")
	}
	if args.ErrorMaxBufferNumber == 0 {
		return errors.NewIllegalParameterError("zero max error buffer number")
	}
	return nil
}

// Same 用于判断两个请求相关的参数容器是否相同。
func (args *RequestArgs) Same(another *RequestArgs) bool {
	if another == nil {
		return false
	}
	if another.MaxDepth != args.MaxDepth {
		return false
	}
	anotherDomains := another.AcceptedDomains
	anotherDomainsLen := len(anotherDomains)
	if anotherDomainsLen != len(args.AcceptedDomains) {
		return false
	}
	if anotherDomainsLen > 0 {
		for i, domain := range anotherDomains {
			if domain != args.AcceptedDomains[i] {
				return false
			}
		}
	}
	return true
}

// ModuleArgsSummary 代表组件相关的参数容器的摘要类型。
type ModuleArgsSummary struct {
	DownloaderListSize int `json:"downloader_list_size"`
	AnalyzerListSize   int `json:"analyzer_List_size"`
	PipelineListSize   int `json:"pipeline_list_size"`
}


func (args *ModuleArgs) Summary() ModuleArgsSummary {
	return ModuleArgsSummary{
		DownloaderListSize: len(args.Downloaders),
		AnalyzerListSize:   len(args.Analyzers),
		PipelineListSize:   len(args.Pipelines),
	}
}




