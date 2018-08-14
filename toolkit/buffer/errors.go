package buffer

import "errors"

//缓冲器和缓冲池中的错误

var ErrClosedBuffer = errors.New("closed buffer")

var ErrClosedPool = errors.New( "closed pool")
