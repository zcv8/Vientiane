package structure

type Item map[string]interface{}

//实现Data接口:判断条目是否有效
func(item *Item)Valid()bool {
	return item != nil
}