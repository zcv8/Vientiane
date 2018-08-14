package reader

import (
	"io"
	"io/ioutil"
	"fmt"
	"bytes"
)

//多重读取器
//主要解决实现io.Reader接口的数据类型只能读取一遍数据的问题

//多重读取器接口
type MultipleReader interface {
	//用于获得一个可关闭的读取器实例
	Reader() io.ReadCloser
}



//多重读取器接口的实现类型
type vientianeMultipleReader struct {
	data []byte
}

func(reader *vientianeMultipleReader)Reader() io.ReadCloser{
	return ioutil.NopCloser(bytes.NewReader(reader.data))
}

func NewMultipleReader(reader io.Reader)(MultipleReader,error) {
	var data []byte
	var err error
	if reader != nil {
		data, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("multiple reader: couldn`t create a new one:%s", err)
		}
	} else {
		data = []byte{}
	}
	return &vientianeMultipleReader{
		data: data,
	}, nil
}