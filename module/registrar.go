package module

import (
	"sync"
	"github.com/Vientiane/errors"
	"fmt"
)

//组件注册器接口
type Registrar interface {
	//用于注册组件实例
	Register(module Module) (bool, error)
	//用于注销组件实例
	Unregister(mid MID) (bool, error)
	//用于获取一个指定类型的组件实例
	//该函数基于负载均衡策略返回实例
	Get(moduleType Type) (Module, error)
	//用于获取指定类型的所有组件实例
	GetAllByType(moduleType Type) (map[MID]Module, error)
	//用于获取所有组件实例
	GetAll() map[MID]Module
	//清除所有的组件注册纪录
	Clear()
}

//组件注册器接口的实现类型
type vientianeRegister struct {
	//组件类型与对应的实例
	moduleTypeMap map[Type]map[MID]Module
	//组件注册专用的读写锁
	rwlock sync.RWMutex
}

func(register *vientianeRegister) Register(module Module) (bool, error) {
	if module == nil {
		return false, errors.NewIllegalParameterError("nil module instance")
	}
	mid := module.ID()
	parts, err := SplitMID(mid)
	if err != nil {
		return false, err
	}
	moduleType := legalLetterTypeMap[parts[0]]
	if !CheckType(moduleType, module) {
		errMsg := fmt.Sprintf("incorrect module type:%s", moduleType)
		return false, errors.NewIllegalParameterError(errMsg)
	}
	register.rwlock.Lock()
	defer register.rwlock.Unlock()
	modules := register.moduleTypeMap[moduleType]
	if modules == nil {
		modules = map[MID]Module{}
	}
	if _, ok := modules[mid]; ok {
		return false, nil
	}
	modules[mid] = module
	register.moduleTypeMap[moduleType] = modules
	return true, nil
}

func(register *vientianeRegister) Unregister(mid MID) (bool, error) {
	parts, err := SplitMID(mid)
	if err != nil {
		return false, err
	}
	moduleType := legalLetterTypeMap[parts[0]]
	var deleted bool
	register.rwlock.Lock()
	defer register.rwlock.Unlock()
	if module, ok := register.moduleTypeMap[moduleType]; ok {
		if _, ok := module[mid]; ok {
			delete(module, mid)
			deleted = true
		}
	}
	return deleted, nil
}

//Get用于获取一个指定类型的实例，会基于负载均衡策略返回实例
func(register *vientianeRegister)Get(moduleType Type) (Module, error) {
	modules, err := register.GetAllByType(moduleType)
	if err != nil {
		return nil, err
	}
	minScore := uint64(0)
	var selectedModule Module
	for _, module := range modules {
		SetScore(module)
		score := module.Score()
		if minScore == 0 || score < minScore {
			selectedModule = module
			minScore = score
		}
	}
	return selectedModule, nil
}

func(register *vientianeRegister)GetAllByType(moduleType Type) (map[MID]Module, error) {
	if !LegalType(moduleType) {
		errMsg := fmt.Sprintf("illegal module type: %s", moduleType)
		return nil, errors.NewIllegalParameterError(errMsg)
	}
	register.rwlock.RLock()
	defer register.rwlock.RUnlock()
	modules:= register.moduleTypeMap[moduleType]
	if len(modules)==0 {
		return nil, ErrNotFoundModuleInstance
	}
	result := map[MID]Module{}
	for mid, module := range modules {
		result[mid] = module
	}
	return result, nil
}

func(register *vientianeRegister)GetAll()map[MID]Module {
	result := map[MID]Module{}
	register.rwlock.RLock()
	defer register.rwlock.RUnlock()
	for _, modules := range register.moduleTypeMap {
		if len(modules) != 0 {
			for mid, module := range modules {
				result[mid] = module
			}
		}
	}
	return result
}

func(register *vientianeRegister)Clear() {
	register.rwlock.Lock()
	defer register.rwlock.Unlock()
	register.moduleTypeMap = map[Type]map[MID]Module{}
}

func NewRegister() Registrar{
	return &vientianeRegister{
		moduleTypeMap: map[Type]map[MID]Module{},
	}
}




