package structure

//数据的接口类型
type Data interface {
	//用于判断数据是否有效
	Valid() bool
}
