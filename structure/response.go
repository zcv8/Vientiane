package structure

import "net/http"

//用于响应的数据结构
type Response struct {
	//Http响应
	httpResp *http.Response
	//响应的深度
	depth uint32
}

//用于获取的http响应
func (resp *Response) HTTPResp() *http.Response {
	return resp.httpResp
}

//用于获取响应的深度
func (resp *Response) Depth() uint32 {
	return resp.depth
}

//实现Data接口:判断响应是否有效
func (resp *Response) Valid() bool {
	return resp.httpResp != nil && resp.httpResp.Body != nil
}

//创建一个响应
func NewResponse(resp *http.Response, depth uint32) *Response {
	return &Response{httpResp: resp, depth: depth}
}
