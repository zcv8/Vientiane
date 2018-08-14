package module

import (
	"github.com/Vientiane/structure"
	"net/http"
)

//组件摘要结构的类型
type SummaryStruct struct {
	ID        MID         `json:"id"`
	Called    uint64      `json:"called"`
	Accepted  uint64      `json:"accepted"`
	Completed uint64      `json:"completed"`
	Handling  uint64      `json:"handling"`
	Extra     interface{} `json:"extra,omitempty"`
}

//用于汇集组件内部的计数类型
type Counts struct {
	CalledCount    uint64
	AcceptedCount  uint64
	CompletedCount uint64
	HandlingNumber uint64
}

//Module代表组件的基础接口类型
//该接口的实现类型必须是并发安全的
type Module interface {
	//用于获取当前组件的ID
	ID() MID
	//用于获取当前组件的网络地址的字符串形式
	Addr() string
	//用于获取当前组件的评分
	Score() uint64
	//用于设置当前组件的评分
	SetScore(score uint64)
	//用于获取评分计算器
	ScoreCalculator() CalculateScore
	//用于获取当前组件被调用的次数
	CalledCount() uint64
	//用于获取当前组件接受的调用计数
	//组件一般会由于超负荷或参数有误而拒绝调用
	AcceptedCount() uint64
	//用于获取当前组件已成功完成的调用的计数
	CompletedCount() uint64
	//用于获取当前组件正在处理的调用数量
	HandlingNumber() uint64
	//用于一次性获取所有计数
	Counts() Counts
	//用于获取组件的摘要
	Summary() SummaryStruct
}

//下载器
//该接口的实现类型必须是并发安全的
type Downloader interface {
	Module
	//根据请求内容获取响应
	Download(req *structure.Request) (*structure.Response, error)
}

//用于解析HTTP响应函数的类型
type ParseResponse func(httpResp *http.Response, respDepth uint32) ([]structure.Data, []error)

//Analyzer代表分析器的接口类型
//该接口的实现类型必须是并发安全的
type Analyzer interface {
	Module
	//用于返回当前分析器使用的响应解析函数的列表
	RespParsers() []ParseResponse
	//根据规则分析响应并返回请求和条目
	Analyze(resp *structure.Response) ([]structure.Data, []error)
}

//用于处理条目的函数类型
type ProcessItem func(item structure.Item)(result structure.Item,err error)

//pipeline代表条目处理管道的接口类型
//该接口的实现类型必须是并发安全的
type Pipeline interface {
	Module
	//用于返回当前条目处理管道使用的条目处理函数列表
	ItemProcessors()[]ProcessItem
	//向条目处理管道发送条目
	//条目依次经过若干条目处理函数的处理
	Send(item structure.Item)[]error
	//返回该处理条目是否是快速处理失败的，也就是在管道中的某个步骤出错，后续步骤就不会执行
	FailFast()bool
	//设置是否快速失败
	SetFailFast(failFast bool)
}


