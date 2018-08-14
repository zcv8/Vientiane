package module

//组件的类型
type Type string

const (
	//下载器
	TYPE_DOWNLOADER Type = "downloader"
	//分析器
	TYPE_ANALYZER Type = "analyzer"
	//条目处理管道
	TYPE_PIPELINE Type = "pipeline"
)

//合法的组件类型和字母的映射
var legalTypeLetterMap = map[Type]string{
	TYPE_DOWNLOADER: "D",
	TYPE_ANALYZER:   "A",
	TYPE_PIPELINE:   "P",
}

//合法的字母和组件类型的映射
var legalLetterTypeMap = map[string]Type{
	"D": TYPE_DOWNLOADER,
	"A": TYPE_ANALYZER,
	"P": TYPE_PIPELINE,
}

//用于判断组件的实例类型是否匹配
func CheckType(moduleType Type, module Module) bool {
	if moduleType == "" || module == nil {
		return false
	}
	switch moduleType {
	case TYPE_DOWNLOADER:
		if _, ok := module.(Downloader); ok {
			return true
		}
	case TYPE_ANALYZER:
		if _, ok := module.(Analyzer); ok {
			return true
		}
	case TYPE_PIPELINE:
		if _, ok := module.(Pipeline); ok {
			return true
		}
	}
	return false
}

// LegalType 用于判断给定的组件类型是否合法。
func LegalType(moduleType Type) bool {
	if _, ok := legalTypeLetterMap[moduleType]; ok {
		return true
	}
	return false
}
