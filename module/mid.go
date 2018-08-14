package module



// DefaultSNGen 代表默认的组件序列号生成器。
//var DefaultSNGen = New(1, 0)

//代表组件ID的模板。
var midTemplate = "%s%d|%s"

//组件ID的模版
type MID string

func SplitMID(mid MID)([]string,error) {
	return nil, nil
}