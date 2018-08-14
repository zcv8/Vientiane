package scheduler

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

