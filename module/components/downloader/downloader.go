package downloader

import (
	"github.com/Vientiane/structure"
	"net/http"
	"github.com/Vientiane/errors"
	"github.com/Vientiane/module"
	"github.com/Vientiane/module/stub"
)


//下载器接口的实现类型
type vientianeDownloader struct{
	//组件的基础实例
	stub.ModuleInternal
	//下载用的http客户端
	httpClient http.Client
}

func(d *vientianeDownloader)Download(req *structure.Request) (*structure.Response, error) {
	d.ModuleInternal.IncrHandlingNumber()
	defer d.ModuleInternal.DecrHandlingNumber()
	d.ModuleInternal.IncrCalledCount()
	if req == nil {
		return nil, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_DOWNLOADER,
			errors.NewIllegalParameterError("nil request"))
	}
	httpReq := req.HTTPReq()
	if httpReq == nil {
		return nil, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_DOWNLOADER,
			errors.NewIllegalParameterError("nil Http request"))
	}
	d.ModuleInternal.IncrAcceptedCount()
	httpResp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	d.ModuleInternal.IncrCompletedCount()
	return structure.NewResponse(httpResp, req.Depth()), nil
}

func NewDownloader(mid module.MID,client *http.Client,
	scoreCalculator module.CalculateScore)(module.Downloader,error) {
	moduleBase, err := stub.NewModuleInternal(mid, scoreCalculator)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.NewCrawlerErrorBy(errors.ERROR_TYPE_DOWNLOADER,
			errors.NewIllegalParameterError("nil http client"))
	}
	return &vientianeDownloader{
		ModuleInternal: moduleBase,
		httpClient:         *client,
	}, nil
}