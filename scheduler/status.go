package scheduler

import (
	"sync"
	"github.com/Vientiane/errors"
	"fmt"
)

type  Status int
//调度器的状态
const(
	//未初始化状态
	SCHED_STATUS_UNINITIALIZED Status = iota
	//正在初始化状态
	SCHED_STATUS_INITIALIZING
	//已初始化状态
	SCHED_STATUS_INITIALIZED
	//正在启动状态
	SCHED_STATUS_STARTING
	//已启动状态
	SCHED_STATUS_STARTED
	//正在停止状态
	SCHED_STATUS_STOPPING
	//已停止状态
	SCHED_STATUS_STOPPED
)

//检查规则
//  1. 处于正在初始化，正在启动或正在停止状态时，不能从外部改变状态
//	2. 想要的状态只能是正在初始化，正在启动或正在停止状态中的一个
//	3. 处于未初始化状态时，不能变为正在启动或正在停止状态
//  4. 处于已启动状态时，不能变为正在初始化或正在启动状态
//  5. 只要未处于已启动状态，就不能变为正在停止状态
func checkStatus(currentStatus Status,wantedStatus Status,lock sync.Locker)(err error) {
	if lock != nil {
		lock.Lock()
		defer lock.Unlock()
	}
	switch currentStatus {
	case SCHED_STATUS_INITIALIZING:
		err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
			"the scheduler is being initialized!")
	case SCHED_STATUS_STARTING:
		err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
			"the scheduler is being started!")
	case SCHED_STATUS_STOPPING:
		err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
			"the scheduler is being stopped!")
	}
	if err != nil {
		return
	}
	if currentStatus == SCHED_STATUS_UNINITIALIZED && (
		wantedStatus == SCHED_STATUS_STARTING || wantedStatus == SCHED_STATUS_STOPPING) {
		err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
			"the scheduler has not yet been initialized!")
	}
	switch wantedStatus {
	case SCHED_STATUS_INITIALIZING:
		switch currentStatus {
		case SCHED_STATUS_STARTED:
			err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
				"the scheduler has been started!")
		}
	case SCHED_STATUS_STARTING:
		switch currentStatus {
		case SCHED_STATUS_UNINITIALIZED:
			err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
				"the scheduler has not been initialized!")
		case SCHED_STATUS_STARTED:
			err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
				"the scheduler has been started!")
		}
	case SCHED_STATUS_STOPPING:
		if currentStatus != SCHED_STATUS_STARTED {
			err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER,
				"the scheduler has not been started!")
		}
	default:
		errMsg :=
			fmt.Sprintf("unsupported wanted status for check! (wantedStatus: %d)",
				wantedStatus)
		err = errors.NewCrawlerError(errors.ERROR_TYPE_SCHEDULER, errMsg)
	}
	return
}

// GetStatusDescription 用于获取状态的文字描述。
func GetStatusDescription(status Status) string {
	switch status {
	case SCHED_STATUS_UNINITIALIZED:
		return "uninitialized"
	case SCHED_STATUS_INITIALIZING:
		return "initializing"
	case SCHED_STATUS_INITIALIZED:
		return "initialized"
	case SCHED_STATUS_STARTING:
		return "starting"
	case SCHED_STATUS_STARTED:
		return "started"
	case SCHED_STATUS_STOPPING:
		return "stopping"
	case SCHED_STATUS_STOPPED:
		return "stopped"
	default:
		return "unkown"
	}
}
