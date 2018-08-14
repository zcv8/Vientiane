package stub

import (
	"github.com/Vientiane/errors"
	"fmt"
	"sync/atomic"
	"github.com/Vientiane/module"
)

//组件的内部基础接口的类型
//内部组件的作用是改变Module中的状态
type ModuleInternal interface {
	module.Module
	//把调用计数增加1
	IncrCalledCount()
	//把接受计数+1
	IncrAcceptedCount()
	//把成功完成计数+1
	IncrCompletedCount()
	//把实时处理数+1
	IncrHandlingNumber()
	//把实时处理数-1
	DecrHandlingNumber()
	//清空
	Clear()
}

//内部接口的实现类型
type vientianeModuleInternal struct {
	mid             module.MID
	addr            string
	score           uint64
	scoreCalculator module.CalculateScore
	calledCount     uint64
	acceptedCount   uint64
	completedCount  uint64
	handlingNumber  uint64
}

func(mi *vientianeModuleInternal)ID() module.MID{
	return mi.mid
}

func(mi *vientianeModuleInternal)Addr() string{
	return mi.addr
}

func(mi *vientianeModuleInternal)Score() uint64{
	return atomic.LoadUint64(&mi.score)
}

func(mi *vientianeModuleInternal)SetScore(score uint64){
	atomic.StoreUint64(&mi.score,score)
}

func(mi *vientianeModuleInternal)ScoreCalculator() module.CalculateScore{
	return mi.scoreCalculator
}

func(mi *vientianeModuleInternal)CalledCount() uint64{
	return atomic.LoadUint64(&mi.calledCount)
}

func(mi *vientianeModuleInternal)AcceptedCount() uint64{
	return atomic.LoadUint64(&mi.acceptedCount)
}

func(mi *vientianeModuleInternal)CompletedCount() uint64{
	return atomic.LoadUint64(&mi.completedCount)
}

func(mi *vientianeModuleInternal)HandlingNumber() uint64{
	return atomic.LoadUint64(&mi.handlingNumber)
}

func(mi *vientianeModuleInternal)Counts() module.Counts{
	return module.Counts{
		CalledCount:    atomic.LoadUint64(&mi.calledCount),
		AcceptedCount:  atomic.LoadUint64(&mi.acceptedCount),
		CompletedCount: atomic.LoadUint64(&mi.completedCount),
		HandlingNumber: atomic.LoadUint64(&mi.handlingNumber),
	}
}

func(mi *vientianeModuleInternal)Summary() module.SummaryStruct{
	counts := mi.Counts()
	return module.SummaryStruct{
		ID:        mi.ID(),
		Called:    counts.CalledCount,
		Accepted:  counts.AcceptedCount,
		Completed: counts.CompletedCount,
		Handling:  counts.HandlingNumber,
		Extra:     nil,
	}
}


func(mi *vientianeModuleInternal)IncrCalledCount(){
	atomic.AddUint64(&mi.calledCount,uint64(1))
}

func(mi *vientianeModuleInternal)IncrAcceptedCount(){
	atomic.AddUint64(&mi.acceptedCount,uint64(1))
}

func(mi *vientianeModuleInternal)IncrCompletedCount(){
	atomic.AddUint64(&mi.completedCount,uint64(1))
}

func(mi *vientianeModuleInternal)IncrHandlingNumber(){
	atomic.AddUint64(&mi.handlingNumber,uint64(1))
}

func(mi *vientianeModuleInternal)DecrHandlingNumber(){
	atomic.AddUint64(&mi.handlingNumber,^uint64(0))
}

func(mi *vientianeModuleInternal)Clear() {
	atomic.StoreUint64(&mi.calledCount, 0)
	atomic.StoreUint64(&mi.acceptedCount, 0)
	atomic.StoreUint64(&mi.completedCount, 0)
	atomic.StoreUint64(&mi.handlingNumber, 0)
}

func NewModuleInternal(mid module.MID,scoreCalculator module.CalculateScore)(
	ModuleInternal,error) {
	parts, err := module.SplitMID(mid)
	if err != nil {
		return nil, errors.NewIllegalParameterError(
			fmt.Sprintf("illegal ID %q:%s", mid, err))
	}
	return &vientianeModuleInternal{
		mid:             mid,
		addr:            parts[2],
		scoreCalculator: scoreCalculator,
	}, nil
}

