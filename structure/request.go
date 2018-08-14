package structure

import "net/http"

//用于请求的数据结构
type Request struct {
	//HTTP请求
	httpReq *http.Request
	//请求的深度
	depth uint32
}

//用于获取请求的深度
func (req *Request) Depth() uint32 {
	return req.depth
}

//用于获取Http请求
func (req *Request) HTTPReq() *http.Request {
	return req.httpReq
}

//实现Data接口:判断请求是否有效
func (req *Request) Valid() bool {
	return req.httpReq != nil && req.httpReq.URL != nil
}

//创建一个请求
func NewRequest(req *http.Request, depth uint32) *Request {
	return &Request{httpReq: req, depth: depth}
}
