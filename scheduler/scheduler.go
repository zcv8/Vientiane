package scheduler

import (
	"github.com/Vientiane/module"
	"github.com/Vientiane/toolkit/buffer"
	"context"
	"sync"
	"github.com/Vientiane/toolkit/cmap"
	"net/http"
	"fmt"
	"github.com/Vientiane/errors"
	"log"
	"github.com/Vientiane/structure"
	"strings"
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

func(sched *vientianeScheduler)Start(firstHTTPReq *http.Request)(err error) {
	defer func() {
		if p := recover(); p != nil {
			errMsg := fmt.Sprintf("Fatal scheduler error:%s", p)
			err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, errMsg)
			log.Fatal(errMsg)
		}
	}()
	var oldStatus Status
	oldStatus, err = sched.checkAndSetStatus(SCHED_STATUS_STARTING)
	if err!=nil {
		return
	}
	defer func() {
		sched.statusLock.Lock()
		if err != nil {
			sched.status = oldStatus
		} else {
			sched.status = SCHED_STATUS_STARTED
		}
		sched.statusLock.Unlock()
	}()
	if firstHTTPReq == nil {
		err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, "nil first Http request")
		return
	}
	var primaryDomain string
	primaryDomain, err = getPrimaryDomain(firstHTTPReq.Host)
	if err != nil {
		return
	}
	sched.acceptedDomainMap.Put(primaryDomain, struct{}{})
	if err = sched.checkBufferPoolForStart(); err != nil {
		return
	}
	sched.download()
	sched.analyze()
	sched.pick()
	firstReq := structure.NewRequest(firstHTTPReq, 0)
	sched.sendReq(firstReq)
	return nil
}

//从缓冲池拿出请求，处理后放到响应池
func (sched *vientianeScheduler)download(){
	go func(){
		for {
			if sched.cancel() {
				break
			}
			datum, err := sched.reqBufferPool.Get()
			if err != nil {
				log.Println("the request pool was closed Break request reception")
				break
			}
			req, ok := datum.(*structure.Request)
			if !ok {
				errMsg := fmt.Sprintf("incorrect request type:%T", datum)
				sendError(errors.New(errMsg), "", sched.errorBufferPool)
			}
			fmt.Println(req.HTTPReq().URL)
			sched.downloadOne(req)
		}
	}()
}


func(sched *vientianeScheduler)cancel()bool {
	select {
	case <-sched.ctx.Done(): //调用cannelFunc取消函数之后，这个通道里面会有一个值
		return true
	default:
		return false
	}
}

//执行具体的下载，并把响应放入响应缓冲池
func(sched *vientianeScheduler)downloadOne(req *structure.Request) {
	if req == nil {
		return
	}
	if sched.cancel() {
		return
	}
	m,err:=sched.register.Get(module.TYPE_DOWNLOADER)
	if err!=nil || m==nil {
		errMsg := fmt.Sprintf("couldn`t get a downloader:%s", err)
		sendError(errors.New(errMsg), "", sched.errorBufferPool)
		sched.sendReq(req)
		return
	}
	downloader,ok:=m.(module.Downloader)
	if !ok {
		errMsg := fmt.Sprintf("incorrect downloader type:%T (MID:%s)", m, m.ID())
		sendError(errors.New(errMsg),"",sched.errorBufferPool)
		sched.sendReq(req)
		return
	}
	resp,err:=downloader.Download(req)
	if resp!=nil{
		sendResp(resp,sched.respBufferPool)
	}
	if err!=nil {
		sendError(err, m.ID(), sched.errorBufferPool)
	}
}

//从缓冲响应池中取出响应并分析，然后把得出的条目放到响应的缓冲池
func(sched *vientianeScheduler)analyze(){
	go func(){
		for{
			if sched.cancel(){
				break
			}
			datum,err:=sched.respBufferPool.Get()
			if err!=nil{
				log.Printf("the response buffer pool was closed break response reception")
				break
			}
			resp,ok:=datum.(*structure.Response)
			if !ok{
				errMsg:=fmt.Sprintf("incorrect response type:%T",datum)
				sendError(errors.New(errMsg),"",sched.errorBufferPool)
			}
			sched.analyzeOne(resp)
		}
	}()
}

//根据给定的相应执行解析并把结果放到相应的缓冲池
func(sched *vientianeScheduler)analyzeOne(resp *structure.Response){
	if resp==nil{
		return
	}
	if sched.cancel(){
		return
	}
	m,err:=sched.register.Get(module.TYPE_ANALYZER)
	if err!=nil || m==nil{
		errMsg:=fmt.Sprintf("could`t get an analyzer:%s",err)
		sendError(errors.New(errMsg),"",sched.errorBufferPool)
		sendResp(resp,sched.respBufferPool)
		return
	}
	analyzer,ok:=m.(module.Analyzer)
	if !ok{
		errMsg := fmt.Sprintf("incorrect analyzer type:%T (MID:%s)", m, m.ID())
		sendError(errors.New(errMsg),"",sched.errorBufferPool)
		sendResp(resp,sched.respBufferPool)
		return
	}
	dataList,errs:=analyzer.Analyze(resp)
	if dataList!=nil{
		for _,data:=range dataList{
			if data==nil {
				continue
			}
			switch d:=data.(type) { //列出实现该接口的类型，也就是实现Data 接口的类型
			case *structure.Request:
				sched.sendReq(d)
			case structure.Item:
				sendItem(d,sched.itemBufferPool)
			default:
				errMsg:=fmt.Sprintf("Unsupported data type %T! (data:%#v)",d,d)
				sendError(errors.New(errMsg),"",sched.errorBufferPool)
			}
		}
	}
	if errs!=nil{
		for _,err:=range errs{
			fmt.Println(err)
			sendError(err,m.ID(),sched.errorBufferPool)
		}
	}
}


func(sched *vientianeScheduler)pick(){
	go func() {
		for {
			if sched.cancel() {
				break
			}
			datum, err := sched.itemBufferPool.Get()
			if err != nil {
				log.Print("the item buffer pool was closed. Break item reception.")
				break
			}
			item, ok := datum.(structure.Item)
			if !ok{
				errMsg:=fmt.Sprintf("incorrect item type:%T",datum)
				sendError(errors.New(errMsg),"",sched.errorBufferPool)
			}
			sched.pickOne(item)
		}
	}()
}

func (sched *vientianeScheduler) pickOne(item structure.Item) {
	if sched.cancel(){
		return
	}
	m,err:=sched.register.Get(module.TYPE_PIPELINE)
	if err!=nil || m==nil {
		errMsg := fmt.Sprintf("couldn't get a pipeline pipline: %s", err)
		sendError(errors.New(errMsg), "", sched.errorBufferPool)
		sendItem(item, sched.itemBufferPool)
		return
	}
	pipeline, ok := m.(module.Pipeline)
	if !ok {
		errMsg := fmt.Sprintf("incorrect pipeline type: %T (MID: %s)",
			m, m.ID())
		sendError(errors.New(errMsg), m.ID(), sched.errorBufferPool)
		sendItem(item, sched.itemBufferPool)
		return
	}
	errs := pipeline.Send(item)
	if errs != nil {
		for _, err := range errs {
			sendError(err, m.ID(), sched.errorBufferPool)
		}
	}
}

//向请求缓冲池中发送请求，同时过滤掉不满足要求的请求
func(sched *vientianeScheduler)sendReq(req *structure.Request)bool{
	if req==nil{
		return false
	}
	if sched.cancel(){
		return false
	}
	httpReq:=req.HTTPReq()
	if httpReq==nil {
		log.Print("Ignore the request! Its HTTP request is invalid!")
		return false
	}
	reqUrl:=httpReq.URL
	if reqUrl==nil{
		log.Print("Ignore the request! Its URL is invalid!")
		return false
	}
	scheme:=strings.ToLower(reqUrl.Scheme)
	if scheme != "http" && scheme != "https" {
		log.Print("Ignore the request! Its URL scheme is %q, but should be %q or %q. (URL: %s)\n",
			scheme, "http", "https", reqUrl)
		return false
	}
	if v:=sched.urlMap.Get(reqUrl.String());v!=nil {
		log.Print("Ignore the request! Its URL is repeated. (URL: %s)\n", reqUrl)
		return false
	}
	pd, _ := getPrimaryDomain(httpReq.Host)
	if sched.acceptedDomainMap.Get(pd)==nil {
		if pd == "bing.net" {
			panic(httpReq.URL)
		}
		log.Print("Ignore the request! Its host %q is not in accepted primary domain map. (URL: %s)\n",
			httpReq.Host, reqUrl)
		return false
	}
	if req.Depth()> sched.maxDepth{
		log.Print("Ignore the request! Its depth %d is greater than %d. (URL: %s)\n",
			req.Depth(), sched.maxDepth, reqUrl)
		return false
	}
	go func(req *structure.Request){
		if err:=sched.reqBufferPool.Put(req);err!=nil{
			log.Print("The request buffer pool was closed. Ignore request sending.")
		}
	}(req)
	sched.urlMap.Put(reqUrl.String(), struct {}{})
	return true
}

// resetContext 用于重置调度器的上下文。
func (sched *vientianeScheduler) resetContext() {
	sched.ctx, sched.cancelFunc = context.WithCancel(context.Background())
}

//停止调度器
func (sched *vientianeScheduler) Stop() (err error) {
	var oldStatus Status
	oldStatus, err = sched.checkAndSetStatus(SCHED_STATUS_STOPPING)
	if err != nil {
		return
	}
	defer func() {
		sched.statusLock.Lock()
		if err != nil {
			sched.status = oldStatus
		} else {
			sched.status = SCHED_STATUS_STOPPED
		}
		sched.statusLock.Unlock()
	}()
	sched.cancelFunc()
	sched.respBufferPool.Close()
	sched.reqBufferPool.Close()
	sched.itemBufferPool.Close()
	sched.errorBufferPool.Close()
	log.Print("Scheduler has been stopped.")
	return nil
}

func(sched *vientianeScheduler)ErrorChan()<-chan error {
	errBuffer := sched.errorBufferPool
	errCh := make(chan error, errBuffer.BufferCap())
	go func(errorBuffer buffer.Pool, errCh chan error) {
		//这里传参是为了防止外部参数失效，参考地址：https://golang.org/doc/go1.8
		for {
			if sched.cancel() {
				close(errCh)
			}
			datum, err := errorBuffer.Get()
			if err != nil {
				log.Print("The error buffer pool was closed. Break error reception.")
				close(errCh)
				break
			}
			err, ok := datum.(error)
			if !ok {
				errMsg := fmt.Sprintf("incorrect error type: %T", datum)
				sendError(errors.New(errMsg), "", sched.errorBufferPool)
				continue
			}
			if sched.cancel() {
				close(errCh)
				break
			}
			errCh <- err
		}
	}(errBuffer, errCh)
	return errCh
}

//判断调度器是否是空闲状态
func(sched *vientianeScheduler)Idle() bool {
	moduleMap := sched.register.GetAll()
	for _, module := range moduleMap {
		//判断各个实例内是否还有正在处理的数据
		if module.HandlingNumber() > 0 {
			return false
		}
	}
	//判断缓冲池中是否还有数据，不用判断错误缓冲池
	if sched.reqBufferPool.Total() > 0 || sched.respBufferPool.Total() > 0 ||
		sched.itemBufferPool.Total() > 0 {
		return false
	}
	return true
}

func (sched *vientianeScheduler) Summary() SchedSummary {
	return sched.summary
}

func (sched *vientianeScheduler) Status() Status {
	var status Status
	sched.statusLock.RLock()
	status = sched.status
	sched.statusLock.RUnlock()
	return status
}

//检查状态是否合法，在合法的情况下设置状态
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

//按照给定的参数初始化缓冲池
//如果某个缓冲池可用并且未关闭，就关闭该缓冲池
func (sched *vientianeScheduler) initBufferPool(dataArgs DataArgs) {
	//初始化请求缓冲池
	if sched.reqBufferPool!=nil&&!sched.reqBufferPool.Closed(){
		sched.reqBufferPool.Close()
	}
	sched.reqBufferPool,_=buffer.NewPool(dataArgs.ReqBufferCap,dataArgs.ReqMaxBufferNumber)
	log.Printf("-- Request buffer pool: bufferCap: %d, maxBufferNumber: %d",
		sched.reqBufferPool.BufferCap(), sched.reqBufferPool.MaxBufferNumber())
	//初始化响应缓冲池
	if sched.respBufferPool!=nil&&!sched.respBufferPool.Closed(){
		sched.respBufferPool.Close()
	}
	sched.respBufferPool,_=buffer.NewPool(dataArgs.RespBufferCap,dataArgs.RespMaxBufferNumber)
	log.Printf("-- Response buffer pool: bufferCap: %d, maxBufferNumber: %d",
		sched.respBufferPool.BufferCap(), sched.respBufferPool.MaxBufferNumber())
	//初始化条目缓冲池
	if sched.itemBufferPool != nil && !sched.itemBufferPool.Closed() {
		sched.itemBufferPool.Close()
	}
	sched.itemBufferPool, _ = buffer.NewPool(
		dataArgs.ItemBufferCap, dataArgs.ItemMaxBufferNumber)
	log.Printf("-- Item buffer pool: bufferCap: %d, maxBufferNumber: %d",
		sched.itemBufferPool.BufferCap(), sched.itemBufferPool.MaxBufferNumber())
	//初始化错误缓冲池
	if sched.errorBufferPool != nil && !sched.errorBufferPool.Closed() {
		sched.errorBufferPool.Close()
	}
	sched.errorBufferPool, _ = buffer.NewPool(
		dataArgs.ErrorBufferCap, dataArgs.ErrorMaxBufferNumber)
	log.Printf("-- Request buffer pool: bufferCap: %d, maxBufferNumber: %d",
		sched.errorBufferPool.BufferCap(), sched.errorBufferPool.MaxBufferNumber())
}

//检查缓冲池是否已为调度器准备就绪
//如果某个缓冲池已经不可用，直接返回错误报告此情况
//如果某个缓冲池已经关闭，按照原先的参数重新初始化它
func (sched *vientianeScheduler) checkBufferPoolForStart() error {
	//检查请求缓冲池
	if sched.reqBufferPool == nil {
		return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
			"nil request buffer pool")
	}
	if sched.reqBufferPool != nil && sched.reqBufferPool.Closed() {
		sched.reqBufferPool, _ = buffer.NewPool(sched.reqBufferPool.BufferCap(),
			sched.reqBufferPool.BufferNumber())
	}
	//检查响应缓冲池
	if sched.respBufferPool == nil {
		return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
			"nil response buffer pool")
	}
	if sched.respBufferPool != nil && sched.respBufferPool.Closed() {
		sched.respBufferPool, _ = buffer.NewPool(sched.respBufferPool.BufferCap(),
			sched.respBufferPool.BufferNumber())
	}
	//检查条目缓冲池
	if sched.itemBufferPool == nil {
		return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
			"nil item buffer pool")
	}
	if sched.itemBufferPool != nil && sched.itemBufferPool.Closed() {
		sched.itemBufferPool, _ = buffer.NewPool(sched.itemBufferPool.BufferCap(),
			sched.itemBufferPool.BufferNumber())
	}
	//检查错误缓冲池
	if sched.errorBufferPool == nil {
		return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
			"nil error buffer pool")
	}
	if sched.errorBufferPool != nil && sched.errorBufferPool.Closed() {
		sched.errorBufferPool, _ = buffer.NewPool(sched.errorBufferPool.BufferCap(),
			sched.errorBufferPool.BufferNumber())
	}
	return nil
}

//注册所有组件
func (sched *vientianeScheduler) registerModules(moduleArgs ModuleArgs) error {
	//注册下载器类型的组件
	for _, d := range moduleArgs.Downloaders {
		if d == nil {
			continue
		}
		ok, err := sched.register.Register(d)
		if err != nil {
			return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, err.Error())
		}
		if !ok {
			errMsg := fmt.Sprintf("Couldn't register downloader instance with MID %q!", d.ID())
			return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, errMsg)
		}
	}
	log.Print("All downloads have been registered. (number: %d)",
		len(moduleArgs.Downloaders))
	//注册分析器类型的组件
	for _, a := range moduleArgs.Analyzers {
		if a == nil {
			continue
		}
		ok, err := sched.register.Register(a)
		if err != nil {
			return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, err.Error())
		}
		if !ok {
			errMsg := fmt.Sprintf("Couldn't register analyzer instance with MID %q!", a.ID())
			return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, errMsg)
		}
	}
	log.Print("All analyzes have been registered. (number: %d)",
		len(moduleArgs.Analyzers))
	//注册处理管道类型的组件
	for _, p := range moduleArgs.Pipelines {
		if p == nil {
			continue
		}
		ok, err := sched.register.Register(p)
		if err != nil {
			return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, err.Error())
		}
		if !ok {
			errMsg := fmt.Sprintf("Couldn't register pipeline instance with MID %q!", p.ID())
			return errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, errMsg)
		}
	}
	log.Print("All pipelines have been registered. (number: %d)",
		len(moduleArgs.Pipelines))
	return nil
}

//向错误缓冲池发送值（进一步加工错误）
func sendError(err error,mid module.MID,
	errorBufferPool buffer.Pool)bool {
	if err == nil || errorBufferPool == nil || errorBufferPool.Closed() {
		return false
	}
	var crawelError errors.CrawlerError
	var ok bool
	crawelError, ok = err.(errors.CrawlerError)
	if !ok {
		var moduleType module.Type
		var errorType errors.ErrorType
		ok, moduleType = module.GetType(mid)
		if !ok {
			errorType = errors.ERROR_TYPE_SCHEDULER
		} else {
			switch moduleType {
			case module.TYPE_DOWNLOADER:
				errorType = errors.ERROR_TYPE_DOWNLOADER
			case module.TYPE_ANALYZER:
				errorType = errors.ERROR_TYPE_ANALYZER
			case module.TYPE_PIPELINE:
				errorType = errors.ERROR_TYPE_PIPELINE
			}
		}
		crawelError = errors.NewCrawlerError(errorType, err.Error())
	}
	if errorBufferPool.Closed() {
		return false
	}
	go func(crawelError errors.CrawlerError) {
		if err := errorBufferPool.Put(crawelError); err != nil {
			log.Printf("the error buffer pool was closed ignore error sending")
		}
	}(crawelError)
	return true
}

//用于向缓冲池发送响应
func sendResp(resp *structure.Response,respBufferPool buffer.Pool)bool {
	if resp == nil || respBufferPool == nil || respBufferPool.Closed() {
		return false
	}
	go func(resp *structure.Response) {
		if err := respBufferPool.Put(resp); err != nil {
			log.Printf("the response buffer pool was closed. ignore response sending")
		}
	}(resp)
	return true
}

// sendItem 会向条目缓冲池发送条目。
func sendItem(item structure.Item, itemBufferPool buffer.Pool) bool {
	if item == nil || itemBufferPool == nil || itemBufferPool.Closed() {
		return false
	}
	go func(item structure.Item) {
		if err := itemBufferPool.Put(item); err != nil {
			log.Print("The item buffer pool was closed. Ignore item sending.")
		}
	}(item)
	return true
}

//因为这个包是调度器的专属包，所以不需要后缀
func New() Scheduler {
	return &vientianeScheduler{}
}


