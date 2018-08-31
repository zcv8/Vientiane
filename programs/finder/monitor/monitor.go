package monitor

import (
	"github.com/Vientiane/scheduler"
	"time"
	"github.com/Vientiane/errors"
	"fmt"
	"golang.org/x/net/context"
	"runtime"
	"encoding/json"
)

//用于监控调度器
//参数：scheduler 代表作为监控目标的调度器
//参数：checkInterval 代表监控时间间隔，单位：纳秒
//参数：summarizeInterval 代表摘要获取的时间间隔 单位：纳秒
//参数：maxIdleCount 代表最大空闲计数
//参数：autoStop 用来指示该方法是否在调度器空闲足够长的时间之后自行停止调度器
//当监控器结束之后，该方法会向作为唯一结果值的通道发送一个代表空闲状态检查次数的数值
func Monitor(
	sched scheduler.Scheduler,
	checkInterval time.Duration,
	summarizeInterval time.Duration,
	maxIdleCount uint,
	autoStop bool)<-chan uint64 {
	//防止调度器不可用
	if sched == nil {
		panic(errors.New("the scheduler is invalid"))
	}
	//防止过小的检查间隔时间对爬取流程造成不良影响
	if checkInterval < time.Millisecond*100 {
		checkInterval = time.Millisecond * 100
	}
	//防止过小的摘要获取时间间隔对爬取流程造成不良影响
	if summarizeInterval < time.Second {
		summarizeInterval = time.Second
	}
	//防止过小的最大空闲计数造成调度器的过早停止
	if maxIdleCount < 10 {
		maxIdleCount = 10
	}
	fmt.Printf("Monitor parameters: checkInterval: %s, summarizeInterval: %s,"+
		" maxIdleCount: %d, autoStop: %v \n",
		checkInterval, summarizeInterval, maxIdleCount, autoStop)
	//生成监控停止通知器
	stopNotifier, stopFunc := context.WithCancel(context.Background())
	//接收和报告错误
	reportError(sched, stopNotifier)
	//纪录摘要信息
	recordSummary(sched, summarizeInterval, stopNotifier)
	//检查计数通道
	checkCountChan := make(chan uint64, 2)
	//检查空闲状态
	checkStatus(sched, checkInterval, maxIdleCount, autoStop, checkCountChan, stopFunc)
	return checkCountChan
}

// summary 代表监控结果摘要的结构。
type summary struct {
	// NumGoroutine 代表Goroutine的数量。
	NumGoroutine int `json:"goroutine_number"`
	// SchedSummary 代表调度器的摘要信息。
	SchedSummary scheduler.SummaryStruct `json:"sched_summary"`
	// EscapedTime 代表从开始监控至今流逝的时间。
	EscapedTime string `json:"escaped_time"`
}

//用来检查状态，并在满足持续空闲时间的条件时采取必要的措施
func checkStatus(sched scheduler.Scheduler,checkInterval time.Duration,
	maxIdleCount uint,autoStop bool,checkCountChan chan<-uint64,stopFunc context.CancelFunc){
		go func() {
			var checkCount uint64
			defer func() {
				stopFunc()
				checkCountChan<-checkCount
			}()
			waitForSchedulerStart(sched)
			var idleCount uint
			var firstIdleTime time.Time
			for{
				//检查调度器的空闲状态
				if sched.Idle(){
					idleCount++
					if idleCount==1{
						firstIdleTime=time.Now()
					}
					if idleCount >=maxIdleCount{
						msg:=fmt.Sprintf("The scheduler has been idle for a period of time" +
							" (about %s)." + " Consider to stop it now.", time.Since(firstIdleTime).String())
						fmt.Println(msg)
						//再次检查调度器的空闲状态，确保他已经可以被停止
						if sched.Idle(){
							if autoStop{
								var result string
								if err:=sched.Stop();err==nil{
									result="success"
								}else{
									result = fmt.Sprintf("failing(%s)", err)
								}
								msg := fmt.Sprintf("Stop scheduler...%s.", result)
								fmt.Println(msg)
							}
							break
						}else {
							if idleCount > 0 {
								idleCount = 0
							}
						}
					}
				}else{
					if idleCount > 0 {
						idleCount = 0
					}
				}
				checkCount++
				time.Sleep(checkInterval)
			}
		}()

}

//用于纪录的摘要信息
func recordSummary(sched scheduler.Scheduler,summarizeInterval time.Duration,stopNotifier context.Context){
	go func() {
		var prevSchedSummaryStruct scheduler.SummaryStruct
		var prevNumGoroutine int
		var recordCount uint64=1
		startTime:=time.Now()
		//等待调度器开始
		waitForSchedulerStart(sched)
		for{
			select {
			case <-stopNotifier.Done():
				return
			default:
			}
			currNumGoroutine:=runtime.NumGoroutine()
			currSchedSummaryStruct:=sched.Summary().Struct()
			// 比对前后两份摘要信息的一致性。只有不一致时才会记录
			if currNumGoroutine != prevNumGoroutine ||
				!currSchedSummaryStruct.Same(prevSchedSummaryStruct) {
				// 记录摘要信息。
				summay := summary{
					NumGoroutine: runtime.NumGoroutine(),
					SchedSummary: currSchedSummaryStruct,
					EscapedTime:  time.Since(startTime).String(),
				}
				b, err := json.MarshalIndent(summay, "", "    ")
				if err != nil {
					fmt.Printf("An error occurs when generating scheduler summary: %s\n", err)
					continue
				}
				_= fmt.Sprintf("Monitor summary[%d]:\n%s", recordCount, b)
				//fmt.Print(msg)
				prevNumGoroutine = currNumGoroutine
				prevSchedSummaryStruct = currSchedSummaryStruct
				recordCount++
			}
			time.Sleep(summarizeInterval)
		}
	}()
}

func reportError(sched scheduler.Scheduler,stopNotifier context.Context){
	go func(){
		//等待调度器开始
		waitForSchedulerStart(sched)
		errChan:=sched.ErrorChan()
		for {
			select {
			case <-stopNotifier.Done():
				return
			default:
			}
			err, ok := <-errChan
			if ok {
				errMsg := fmt.Sprintf("Received an error from error channel: %s", err)
				fmt.Printf(errMsg)
			}
			time.Sleep(time.Microsecond)
		}
	}()
}

func waitForSchedulerStart(sched scheduler.Scheduler) {
	for sched.Status() != scheduler.SCHED_STATUS_STARTED {
		time.Sleep(time.Microsecond)
	}
}