package pipeline

import (
	"github.com/Vientiane/structure"
	"github.com/Vientiane/errors"
	"fmt"
	"github.com/Vientiane/module/stub"
	"github.com/Vientiane/module"
)

//pipeline接口的实现类型
type vientianePipeline struct{
	stub.ModuleInternal
	//条目处理器列表
	itemProcessors []module.ProcessItem
	//处理是否需要快速失败
	failFast bool
}

func(p *vientianePipeline)ItemProcessors()[]module.ProcessItem{
	processors:=make([]module.ProcessItem,len(p.itemProcessors))
	copy(processors,p.itemProcessors)
	return processors
}

func(p *vientianePipeline)FailFast()bool{
	return p.failFast
}

func(p *vientianePipeline)SetFailFast(failFast bool) {
	p.failFast = failFast
}

func(p *vientianePipeline)Send(item structure.Item)[]error{
	p.ModuleInternal.IncrHandlingNumber()
	defer p.ModuleInternal.DecrHandlingNumber()
	p.ModuleInternal.CalledCount()
	var errs []error
	if item==nil {
		errs = append(errs, errors.NewIllegalParameterError("nil item"))
		return errs
	}
	p.ModuleInternal.IncrAcceptedCount()
	var currentItem = item
	for _,processor:=range p.itemProcessors {
		processedItem, err := processor(currentItem)
		if err!=nil{
			errs=append(errs,err)
			if p.failFast{
				break
			}
		}
		if processedItem!=nil {
			currentItem = processedItem
		}
	}
	if len(errs)==0{
		p.ModuleInternal.IncrCompletedCount()
	}
	return errs
}

func NewPipeLine(mid module.MID,scoreCalculator module.CalculateScore,itemProcessors []module.ProcessItem)(module.Pipeline,error) {
	moduleBase, err := stub.NewModuleInternal(mid, scoreCalculator)
	if err != nil {
		return nil, err
	}
	if itemProcessors == nil {
		return nil, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_PIPELINE,
			errors.NewIllegalParameterError("nil itemProcessors"))
	}
	if len(itemProcessors) == 0 {
		return nil, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_PIPELINE,
			errors.NewIllegalParameterError("empty item processor list"))
	}
	var processors []module.ProcessItem
	for i, processor := range itemProcessors {
		if processor == nil {
			errMsg := fmt.Sprintf("nil item processor[%d]", i)
			return nil, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_PIPELINE,
				errors.NewIllegalParameterError(errMsg))
		}
		processors = append(processors, processor)
	}
	return &vientianePipeline{
		ModuleInternal: moduleBase,
		itemProcessors: itemProcessors,
	}, nil
}
