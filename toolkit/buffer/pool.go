package buffer

import (
	"sync"
	"github.com/Vientiane/errors"
	"fmt"
	"sync/atomic"
)

//缓冲池

//数据缓冲池的接口类型
type Pool interface {
	//用于获取池中缓冲器的统一容量
	BufferCap() uint32
	//用于获取池中缓冲器的最大数量
	MaxBufferNumber() uint32
	//用于获取当前缓冲池中的缓存器的数量
	BufferNumber() uint32
	//用于获取缓冲池中的数据总数
	Total() uint64
	//用于向缓存池中放入数据
	Put(datum interface{})error
	//用于从缓冲池中获取数据
	Get()(datum interface{},err error)
	//关闭缓冲池
	Close()bool
	//判断缓冲池是否已关闭
	Closed() bool
}



//数据缓冲池接口的实现类型
type vientianePool struct {
	//缓冲器的统一容量
	bufferCap uint32
	//缓冲器的最大数量
	maxBufferNumber uint32
	//缓冲器的实际数量
	bufferNumber uint32
	//池中数据的总数
	total uint64
	//存放缓冲器的通道
	//这里采用双层通道设计
	bufCh chan Buffer
	//缓存池的关闭状态：0-未关闭 1-已关闭
	closed uint32
	//保护内部共享资源的读写锁
	rwlock sync.RWMutex
}

func(p *vientianePool)BufferCap() uint32{
	return p.bufferCap
}

func (p *vientianePool)MaxBufferNumber() uint32{
	return p.maxBufferNumber
}

func (p *vientianePool)BufferNumber()uint32{
	return atomic.LoadUint32(&p.bufferNumber)
}

func(p *vientianePool)Total() uint64 {
	return atomic.LoadUint64(&p.total)
}

//这里包含当缓冲池中的缓冲器不够用的时候动态的添加缓冲器
func(p *vientianePool)Put(datum interface{})(err error) {
	if p.Closed() {
		return ErrClosedPool
	}
	var count uint32
	maxCount := p.BufferNumber() * 5
	var ok bool
	for buf := range p.bufCh {
		ok, err = p.putData(buf, datum, &count, maxCount)
		if ok || err != nil {
			break
		}
	}
	return
}

func (p *vientianePool)Close()bool {
	if atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		p.rwlock.Lock()
		defer p.rwlock.Unlock()
		close(p.bufCh)
		for buf := range p.bufCh {
			buf.Close()
		}
		return true
	}
	return false
}

func (p *vientianePool)Closed()bool {
	if atomic.LoadUint32(&p.closed) == 0 {
		return false
	}
	return true
}

func (p *vientianePool)Get()(datum interface{},err error) {
	if p.Closed() {
		return nil, ErrClosedPool
	}
	var count uint32
	maxCount := p.BufferNumber() * 10
	for buf := range p.bufCh {
		datum, err = p.getData(buf, &count, maxCount)
		if datum != nil || err != nil { //报错或者是取到值就返回
			break
		}
	}
	return
}


func(p *vientianePool)getData(buf Buffer,count *uint32,
	maxCount uint32)(datum interface{},err error) {
	if p.Closed(){
		return nil,ErrClosedPool
	}
	defer func(){
		if *count>=maxCount && buf.Len()==0 && p.BufferNumber()>1 {
			buf.Close()
			atomic.AddUint32(&p.bufferNumber, ^uint32(0))
			*count = 0
			return
		}
		p.rwlock.RLock()
		if p.Closed(){
			atomic.AddUint32(&p.bufferNumber,^uint32(0))
			err =  ErrClosedPool
		}else {
			p.bufCh <- buf
		}
		p.rwlock.RUnlock()
	}()
	datum,err = buf.Get()
	if datum !=nil{
		atomic.AddUint64(&p.total,^uint64(0))
		return
	}
	if err!=nil{
		return
	}
	(*count)++
	return
}


func(p *vientianePool) putData(
	buf Buffer,
	datum interface{},
	count *uint32,
	maxCount uint32)(ok bool,err error) {
	if p.Closed() {
		return false, ErrClosedPool
	}
	defer func() {
		p.rwlock.RLock()
		if p.Closed() { //当向buf添加完数据之后，发现缓存池已经关闭，则不会归还缓冲器
			atomic.AddUint32(&p.bufferNumber, ^uint32(0)) //进行-1操作
			err = ErrClosedPool
		} else {
			p.bufCh <- buf
		}
		p.rwlock.RUnlock()
	}()
	ok, err = buf.Put(datum)
	if ok {
		atomic.AddUint64(&p.total, 1)
		return
	}
	if err != nil { //这里证明是报错(缓冲器关闭)，而不是缓冲器已经满了
		return
	}
	//执行到这里说明ok是false，并且err是空，则说明缓冲器已满
	(*count)++
	//如果加入失败的次数达到阈值，并且缓冲池中缓冲器的数量并未达到最大值，那就尝试增加一个缓冲器并加入到缓冲池
	if *count > maxCount && p.BufferNumber() < p.MaxBufferNumber() {
		p.rwlock.Lock()
		if p.BufferNumber() < p.MaxBufferNumber() { //双重检查，当前数量为最大值-1的时候，并发的执行放入缓冲器，会导致阻塞
			if p.Closed() {
				p.rwlock.Unlock()
				return
			}
			newBuf, _ := NewBuffer(p.bufferCap)
			newBuf.Put(datum)
			p.bufCh <- newBuf
			atomic.AddUint32(&p.bufferNumber, 1)
			atomic.AddUint64(&p.total, 1)
			ok = true
		}
		p.rwlock.Unlock()
		*count = 0
	}
	return
}


func NewPool(bufferCap uint32, maxBufferNumber uint32)(Pool,error) {
	if bufferCap == 0 {
		errMsg := fmt.Sprintf("illegal buffer cap for buffer pool: %d", bufferCap)
		return nil, errors.NewIllegalParameterError(errMsg)
	}
	if maxBufferNumber == 0 {
		errMsg := fmt.Sprintf("illegal max buffer number for buffer pool: %d", maxBufferNumber)
		return nil, errors.NewIllegalParameterError(errMsg)
	}
	bufCh := make(chan Buffer, maxBufferNumber)
	buf, _ := NewBuffer(bufferCap)
	bufCh <- buf
	return &vientianePool{
		bufferCap:       bufferCap,
		maxBufferNumber: maxBufferNumber,
		bufCh:           bufCh,
		bufferNumber:    1,
	}, nil
}




