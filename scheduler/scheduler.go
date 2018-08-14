package scheduler

import (
	"github.com/Vientiane/module"
	"github.com/Vientiane/toolkit/buffer"
	"context"
	"sync"
	"github.com/Vientiane/toolkit/cmap"
)

//scheduler接口的实现类型
type vientianeScheduler struct {
	//爬取的最大深度
	maxDepth uint32
	//可以接受的Url的主域名的字典
	acceptedDomainMap cmap.ConcurrentMap
	//组件注册器
	register module.Registrar
	//请求缓冲池
	reqBufferPool buffer.Pool
	//响应缓存池
	respBufferPool buffer.Pool
	//条目的缓冲池
	itemBufferPool buffer.Pool
	//错误缓冲池
	errorBufferPool buffer.Pool
	//已处理的Url字典
	urlMap cmap.ConcurrentMap
	//上下文，用于感知调度器的停止
	ctx context.Context
	//去掉函数，用于停止调度器
	cancelFunc context.CancelFunc
	//状态
	status Status
	//专用于状态的读写锁(非原子操作可以操作的那几个类型)
	statusLock sync.RWMutex
	//摘要信息
	summary SchedSummary
}

func(sched *vientianeScheduler)Init(requestArgs RequestArgs,dataArgs DataArgs,
	moduleArgs ModuleArgs)(err error) {
	//检查状态
	var oldStatus Status
	oldStatus,err = sched.checkAndSetStatus(SCHED_STATUS_INITIALIZING)
	if err!=nil{
		return
	}
	defer func(){
		sched.statusLock.Lock()
		if err!=nil{
			sched.status = oldStatus
		}else {
			sched.status = SCHED_STATUS_INITIALIZED
		}
		sched.statusLock.Unlock()
	}()
	//检查参数
	if err = requestArgs.Check();err!=nil{
		return err
	}
	if err = dataArgs.Check(); err != nil {
		return err
	}
	if err = moduleArgs.Check(); err != nil {
		return err
	}
	if sched.register == nil {
		sched.register = module.NewRegister()
	}else{
		sched.register.Clear()
	}
	sched.maxDepth = requestArgs.MaxDepth
	sched.acceptedDomainMap, _ =
		cmap.NewConcurrentMap(1, nil)
	for _,domain:=range requestArgs.AcceptedDomains {
		sched.acceptedDomainMap.Put(domain, struct {}{})
	}
	sched.urlMap, _ = cmap.NewConcurrentMap(16, nil)
	sched.initBufferPool(dataArgs)
	sched.resetContext()
	sched.summary = newSchedSummary(requestArgs, dataArgs, moduleArgs, sched)
	if err = sched.registerModules(moduleArgs); err != nil {
		return err
	}
	return nil
}

func(sched *vientianeScheduler)checkAndSetStatus(wantedStatus Status)(oldStatus Status,err error) {
	sched.statusLock.Lock()
	defer sched.statusLock.Unlock()
	oldStatus = sched.status
	err = checkStatus(oldStatus, wantedStatus, nil)
	if err == nil {
		sched.status = wantedStatus
	}
	return
}

//因为这个包是调度器的专属包，所以不需要后缀
func New() Scheduler {
	return &vientianeScheduler{}
}


