package module

//用于计算积分的工具/服务

//用于计算组件评分的函数类型
type CalculateScore func(count Counts) uint64

// CalculateScoreSimple 代表简易的组件评分计算函数。
func CalculateScoreSimple(counts Counts) uint64 {
	return counts.CalledCount +
		counts.AcceptedCount<<1 +
		counts.CompletedCount<<2 +
		counts.HandlingNumber<<4
}

func SetScore(module Module) bool {
	calculateScore := module.ScoreCalculator()
	if calculateScore==nil{
		calculateScore = CalculateScoreSimple
	}
	newScore:= calculateScore(module.Counts())
	if newScore == module.Score() {
		return false
	}
	module.SetScore(newScore)
	return true
}




