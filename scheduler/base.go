package scheduler

import (
	"net/http"
)


//调度器接口类型
type Scheduler interface{
	//初始化调度器
	Init(requestArgs RequestArgs,dataArgs DataArgs,moduleArgs ModuleArgs)(err error)
	//用于启动调度器并执行爬取过程
	Start(firstHTTPReq *http.Request)(err error)
	//停止调度器的运行
	Stop()(err error)
	//用于获取调度器的状态
	Status()Status
	//用于获得错误的接收通道
	//调度器以及各个处理模块出现的错误都会发送到这个管道
	//若结果为nil则代表调度器已经停止或者是错误通道不可以
	ErrorChan()<-chan error
	//用于判断所有处理模块是否都处于空闲状态
	Idle()bool
	//用于获取摘要实例
	Summary() SchedSummary
}


