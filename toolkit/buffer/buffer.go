package buffer

import (
	"sync"
	"fmt"
	"github.com/Vientiane/errors"
	"sync/atomic"
)

//缓冲器
//不直接使用通道是因为通道类型无法判断通道是否已关闭，对关闭的通道进行put操作和close操作会导致panic

//FIFO的缓冲器的接口类型
type Buffer interface {
	//用于获取缓冲器的容量
	Cap() uint32
	//用于获取缓冲器中的数据总量
	Len() uint32
	//向缓冲器中放入数据(非阻塞)
	Put(datum interface{})(bool,error)
	//从缓冲器获取数据(非阻塞)
	Get()(interface{},error)
	//关闭缓冲器
	Close()bool
	//用于判断缓冲器是否关闭
	Closed() bool
}

//缓冲器接口的实现类型
type vientianeBuffer struct {
	//存放数据的通道
	ch chan interface{}
	//缓冲器的关闭状态:0-未关闭 1-已关闭
	closed uint32
	//为了消除因关闭缓冲器而产生的竞态条件的读写锁
	closingLock sync.RWMutex
}

func(buf *vientianeBuffer)Cap()uint32 {
	return uint32(cap(buf.ch))
}

func(buf *vientianeBuffer)Len()uint32 {
	return uint32(len(buf.ch))
}

func(buf *vientianeBuffer)Put(datum interface{})(ok bool,err error) {
	//防止在向通道写入数据的时候，被并发的关闭了通道导致panic
	//这里使用读锁，是允许并发的往通道中写入数据
	buf.closingLock.RLock()
	defer buf.closingLock.RUnlock()
	if buf.closed == 1 {
		return false, ErrClosedBuffer
	}
	select {
	case buf.ch <- datum:
		ok = true
	default:
		ok = false //此时error 为 nil 则说明是缓冲器已满，而不是缓冲器被关闭
	}
	return
}

func(buf *vientianeBuffer)Get()(interface{},error){
	select {
	case datum, ok := <-buf.ch:
		if !ok {
			return nil, ErrClosedBuffer
		}
		return datum, nil
	default:
		return nil, nil
	}
}

func(buf *vientianeBuffer)Close()bool {
	if atomic.CompareAndSwapUint32(&buf.closed, 0, 1) {
		buf.closingLock.Lock()
		defer buf.closingLock.Unlock()
		close(buf.ch)
		return true
	}
	return false
}

func(buf *vientianeBuffer)Closed()bool {
	if atomic.LoadUint32(&buf.closed) == 0 {
		return false
	}
	return true
}


//创建一个缓冲器
func NewBuffer(size uint32)(Buffer,error) {
	if size == 0 {
		errMsg := fmt.Sprintf("illegal size for buffer:%d", size)
		return nil, errors.NewIllegalParameterError(errMsg)
	}
	return &vientianeBuffer{
		ch: make(chan interface{}, size),
	}, nil
}


