package analyzer

import (
	"github.com/Vientiane/structure"
	"github.com/Vientiane/errors"
	"fmt"
	"github.com/Vientiane/toolkit/reader"
	"github.com/Vientiane/module"
	"github.com/Vientiane/module/stub"
)

//分析器接口的实现类型
type vientianeAnalyzer struct{
	stub.ModuleInternal
	//响应解析器列表
	respParsers []module.ParseResponse
}

func(a *vientianeAnalyzer)RespParsers() []module.ParseResponse {
	//每个实例拿到的都是函数的拷贝
	parser := make([]module.ParseResponse, len(a.respParsers))
	copy(parser, a.respParsers)
	return parser
}

func(a *vientianeAnalyzer)Analyze(resp *structure.Response) (dataList []structure.Data,errorList []error) {
	a.ModuleInternal.IncrHandlingNumber()
	defer a.ModuleInternal.DecrHandlingNumber()
	a.ModuleInternal.IncrCalledCount()
	if resp == nil {
		errorList = append(errorList, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_ANALYZER,
			errors.NewIllegalParameterError("nil response")))
		return
	}
	httpResp := resp.HTTPResp()
	if httpResp == nil {
		errorList = append(errorList,errors.NewCrawlerErrorBy(errors.ERROR_TYPE_ANALYZER,
			errors.NewIllegalParameterError("nil http response")))
		return
	}
	httpReq := httpResp.Request
	if httpReq == nil {
		errorList = append(errorList, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_ANALYZER,
			errors.NewIllegalParameterError("nil http request")))
		return
	}
	reqUrl := httpReq.URL
	if reqUrl == nil {
		errorList = append(errorList, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_ANALYZER,
			errors.NewIllegalParameterError("nil http request url")))
		return
	}
	a.ModuleInternal.IncrAcceptedCount()
	respDepth := resp.Depth()
	if httpResp.Body != nil {
		defer httpResp.Body.Close()
	}
	multipleReader, err := reader.NewMultipleReader(httpResp.Body)
	if err != nil {
		errorList = append(errorList, errors.NewCrawlerError(errors.ERROR_TYPE_ANALYZER, err.Error()))
		return
	}
	dataList = []structure.Data{}
	for _, respParser := range a.respParsers {
		httpResp.Body = multipleReader.Reader()
		pDataList, pErrorList := respParser(httpResp, respDepth)
		if pDataList != nil {
			for _, pData := range pDataList {
				if pData != nil {
					dataList = appendDataList(dataList, pData, respDepth)
				}
			}
		}
		if pErrorList != nil {
			for _, pError := range pErrorList {
				if pError != nil {
					errorList = append(errorList, pError)
				}
			}
		}
	}
	if len(errorList) == 0 {
		a.ModuleInternal.IncrCompletedCount()
	}
	return dataList, errorList
}

// appendDataList 用于添加请求值或条目值到列表
func appendDataList(dataList []structure.Data, data structure.Data,
	respDepth uint32) []structure.Data {
	if data == nil {
		return dataList
	}
	req, ok := data.(*structure.Request)
	if !ok {
		return append(dataList, data)
	}
	newDepth := respDepth + 1
	if req.Depth() != newDepth {
		req = structure.NewRequest(req.HTTPReq(), newDepth)
	}
	return append(dataList, req)
}

func NewAnalyzer(mid module.MID,scoreCalculator module.CalculateScore,
	respParsers []module.ParseResponse)(module.Analyzer,error) {
	moduleBase, err := stub.NewModuleInternal(mid, scoreCalculator)
	if err != nil {
		return nil, err
	}
	if respParsers == nil {
		return nil, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_ANALYZER,
			errors.NewIllegalParameterError("nil response parsers"))
	}
	if len(respParsers) == 0 {
		return nil, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_ANALYZER,
			errors.NewIllegalParameterError("empty response parser list"))
	}
	var innerParsers []module.ParseResponse
	for i, parser := range respParsers {
		if parser == nil {
			errMsg := fmt.Sprintf("nil response parser[%d]", i)
			return nil, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_ANALYZER,
				errors.NewIllegalParameterError(errMsg))
		}
		innerParsers = append(innerParsers, parser)
	}
	return &vientianeAnalyzer{
		ModuleInternal: moduleBase,
		respParsers:    innerParsers,
	}, nil
}

