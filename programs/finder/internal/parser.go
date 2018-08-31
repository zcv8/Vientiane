package internal

import (
	"github.com/Vientiane/module"
	"net/http"
	"github.com/Vientiane/structure"
	"path"
	"fmt"
	"strings"
	"github.com/PuerkitoBio/goquery"
	"net/url"
)

func genResponseParsers()[]module.ParseResponse{
	//分析函数用来发现的请求
	parserLink:=func(httpResp *http.Response,respDepth uint32)([]structure.Data,[]error) {
		dataList := make([]structure.Data, 0)
		//检查响应
		if httpResp == nil {
			return nil, []error{fmt.Errorf("nil HTTP response")}
		}
		httpReq := httpResp.Request
		if httpReq == nil {
			return nil, []error{fmt.Errorf("nil HTTP request")}
		}
		reqUrl := httpReq.URL
		if httpResp.StatusCode != 200 {
			err := fmt.Errorf("unsupported status code %d (requestURL: %s)",
				httpResp.StatusCode, reqUrl)
			return nil, []error{err}
		}
		body := httpResp.Body
		if body == nil {
			err := fmt.Errorf("nil HTTP response body (requestURL: %s)",
				reqUrl)
			return nil, []error{err}
		}
		//检查Http响应头的内容类型
		var matchedContentType bool
		if httpResp.Header != nil {
			contentTypes := httpResp.Header["Content-Type"]
			for _, ct := range contentTypes {
				if strings.HasPrefix(ct, "text/html") {
					matchedContentType = true
					break
				}
			}
		}
		if !matchedContentType {
			return dataList, nil
		}
		//解析http响应体
		doc, err := goquery.NewDocumentFromReader(body)
		if err != nil {
			return dataList, []error{err}
		}
		errs := make([]error, 0)
		//查找a标签并提取地址
		doc.Find("a").Each(func(index int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if !exists || href == "" || href == "#" || href == "/" {
				return
			}
			href = strings.TrimSpace(href)
			lowerHref := strings.ToLower(href)
			if href == "" || strings.HasPrefix(lowerHref, "javascript") {
				return
			}
			aURL, err := url.Parse(lowerHref)
			if err != nil {
				fmt.Printf("An error occurs when parsing attribute %q in tag %q : %s (href: %s)",
					err, "href", "a", href)
				return
			}
			if aURL.IsAbs() {
				aURL = reqUrl.ResolveReference(aURL)
			}
			httpReq, err := http.NewRequest("GET", aURL.String(), nil)
			if err != nil {
				errs = append(errs, err)
			} else {
				req := structure.NewRequest(httpReq, respDepth)
				dataList = append(dataList, req)
			}
		})
		// 查找img标签并提取地址。
		doc.Find("img").Each(func(index int, sel *goquery.Selection) {
			imgSrc, exists := sel.Attr("src")
			if !exists || imgSrc == "" || imgSrc == "#" || imgSrc == "/" {
				return
			}
			fmt.Println("-----------------------",imgSrc)
			imgSrc = strings.TrimSpace(imgSrc)
			imgURL, err := url.Parse(imgSrc)
			if err != nil {
				errs = append(errs, err)
				return
			}
			if !imgURL.IsAbs() {
				imgURL = reqUrl.ResolveReference(imgURL)
			}
			httpReq, err := http.NewRequest("GET", imgURL.String(), nil)
			if err != nil {
				errs = append(errs, err)
			} else {
				req := structure.NewRequest(httpReq, respDepth)
				dataList = append(dataList, req)
			}
		})
		return dataList, errs
	}
	//分析函数用来对发现的图片进行处理
	parseImage:= func(httpResp *http.Response,respDepth uint32) ([]structure.Data,[]error) {
		// 检查响应。
		if httpResp == nil {
			return nil, []error{fmt.Errorf("nil HTTP response")}
		}
		httpReq := httpResp.Request
		if httpReq == nil {
			return nil, []error{fmt.Errorf("nil HTTP request")}
		}
		reqUrl := httpReq.URL
		if httpResp.StatusCode != 200 {
			err := fmt.Errorf("unsupported status code %d (requestURL: %s)",
				httpResp.StatusCode, reqUrl)
			return nil, []error{err}
		}
		httpRespBody := httpResp.Body
		if httpRespBody == nil {
			err := fmt.Errorf("nil HTTP response body (requestURL: %s)",
				reqUrl)
			return nil, []error{err}
		}
		// 检查HTTP响应头中的内容类型。
		dataList := make([]structure.Data, 0)
		var pictureFormat string
		if httpResp.Header != nil {
			contentTypes := httpResp.Header["Content-Type"]
			var contentType string
			for _, ct := range contentTypes {
				if strings.HasPrefix(ct, "image") {
					contentType = ct
					break
				}
			}
			index1:=strings.Index(contentType,"/")
			index2:=strings.Index(contentType,";")
			if index1>0 {
				if index2 < 0 {
					pictureFormat = contentType[index1+1:]
				} else if index1 < index2 {
					pictureFormat = contentType[index1+1 : index2]
				}
			}
		}
		if pictureFormat=="" {
			return dataList, nil
		}
		item := make(map[string]interface{})
		item["reader"] = httpRespBody
		item["name"] = path.Base(reqUrl.Path)
		item["ext"] = pictureFormat
		dataList = append(dataList, structure.Item(item))
		return dataList, nil
	}

	return []module.ParseResponse{parserLink,parseImage}
}
