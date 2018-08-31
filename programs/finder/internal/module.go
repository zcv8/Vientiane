package internal

import (
	"github.com/Vientiane/module"
	"github.com/Vientiane/toolkit/generator"
	"github.com/Vientiane/module/components/downloader"
	"github.com/Vientiane/module/components/analyzer"
	"github.com/Vientiane/module/components/pipeline"
)
var snGen = generator.NewSNGenertor(1, 0);

//用于获取下载器列表
func GetDownloaders(number uint8)([]module.Downloader,error) {
	downloaders := []module.Downloader{}
	if number == 0 {
		return downloaders, nil
	}
	for i := uint8(0); i < number; i++ {
		mid, err := module.GenMID(module.TYPE_DOWNLOADER, snGen.Get(), nil)
		if err != nil {
			return downloaders, err
		}
		d, err := downloader.NewDownloader(mid, genHttpClient(), module.CalculateScoreSimple)
		if err != nil {
			return downloaders, err
		}
		downloaders = append(downloaders, d)
	}
	return downloaders, nil
}

//用于获取分析器列表
func GetAnalyzers(number uint8)([]module.Analyzer,error){
	analyzers:=[]module.Analyzer{}
	if number==0{
		return analyzers,nil
	}
	for i := uint8(0); i < number; i++ {
		mid, err := module.GenMID(
			module.TYPE_ANALYZER, snGen.Get(), nil)
		if err != nil {
			return analyzers, err
		}
		a, err := analyzer.NewAnalyzer(mid, module.CalculateScoreSimple, genResponseParsers())
		if err != nil {
			return analyzers, err
		}
		analyzers = append(analyzers, a)
	}
	return analyzers, nil
}

//用于获取分析器列表
func GetPipelines(number uint8, dirPath string) ([]module.Pipeline, error) {
	pipelines := []module.Pipeline{}
	if number == 0 {
		return pipelines, nil
	}
	for i := uint8(0); i < number; i++ {
		mid, err := module.GenMID(
			module.TYPE_PIPELINE, snGen.Get(), nil)
		if err != nil {
			return pipelines, err
		}
		a, err := pipeline.NewPipeLine(
			mid, module.CalculateScoreSimple, genItemProcessors(dirPath))
		if err != nil {
			return pipelines, err
		}
		a.SetFailFast(true)
		pipelines = append(pipelines, a)
	}
	return pipelines, nil
}

