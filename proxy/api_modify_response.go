package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/robertkrimen/otto"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ApiRespModifier struct {
	Note      string `json:"note"`
	Enable    bool   `json:"enable"`
	Rule      string `json:"rule"`
	jsRunTime chan otto.Value
}

type RespModifier []*ApiRespModifier

func newRespModifierSlice() RespModifier {
	return make([]*ApiRespModifier, 0)
}

func (rm RespModifier) Init() error {
	for _, item := range rm {
		err := item.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

//ModifierResp 使用动态规则对response进行处理
func (rm RespModifier) ModifierResp(req *http.Request, resp *http.Response) (mod bool, err error) {
	if len(rm) == 0 {
		return
	}
	items := newRespModifierSlice()
	for _, item := range rm {
		if !item.Enable || item.Rule == "" {
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return
	}

	reqMap := make(map[string]interface{})
	reqMap["get"] = req.URL.Query()
	reqMap["method"] = req.Method
	reqMap["request_uri"] = req.RequestURI
	reqMap["path"] = req.URL.Path
	reqMap["remote_addr"] = req.RemoteAddr

	_header := make(map[string]string)
	for k, vs := range req.Header {
		_header[k] = strings.Join(vs, ",")
	}
	reqMap["header"] = _header

	reqJsObj, _ := otto.New().Object(`req={}`)
	for k, v := range reqMap {
		reqJsObj.Set(k, v)
	}
	/////////////////////////////////////////////////
	resJsObj, _ := otto.New().Object(`res={}`)

	resMap := make(map[string]interface{})
	resMap["http_code"] = resp.StatusCode
	resMap["content_length"] = resp.ContentLength

	_res_header := make(map[string]string)
	for k, vs := range resp.Header {
		_res_header[k] = strings.Join(vs, ",")
	}
	resMap["header"] = _res_header
	for k, v := range resMap {
		resJsObj.Set(k, v)
	}
	//==============================================
	modReqBody := url.Values{}
	_bs, _ := json.Marshal(reqMap)
	modReqBody.Add("req", string(_bs))
	_bs_res, _ := json.Marshal(resMap)
	modReqBody.Add("resp", string(_bs_res))
	//==============================================
	hasReadBody := false
	notChangeRespStr := "NOT_CHANGE"
	var rawBodyBf *bytes.Buffer
	for _, item := range items {
		fn := <-item.jsRunTime
		var js_ret otto.Value
		(func() {
			if err_rec := recover(); err_rec != nil {
				log.Println("[recover]", err_rec)
			}
			js_ret, err = fn.Call(fn, reqJsObj, resJsObj)
		})()
		item.jsRunTime <- fn
		
		if err != nil {
			return true, err
		}
		if !js_ret.IsString() {
			continue
		}
		ret_str := js_ret.String()
		ret_str=strings.TrimSpace(ret_str)
		if ret_str == "" {
			continue
		}
		if ret_str == "not" {
			break
		}

		resp.Header.Set("resp_modifier", ret_str)

		_, err = url.Parse(ret_str)
		if err != nil {
			return true, err
		}
		if !hasReadBody {
			rawBodyBf = forgetRead(&resp.Body)
			hasReadBody = true
			modReqBody.Add("resp_content", rawBodyBf.String())
		}

		myReq, err := http.NewRequest("POST", ret_str, strings.NewReader(modReqBody.Encode()))
		if err != nil {
			return true, err
		}
		client := &http.Client{
			Timeout: 500 * time.Millisecond,
		}
		myReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		newRes, err := client.Do(myReq)
		if err != nil {
			return true, err
		}
		defer newRes.Body.Close()
		newBodyBs, err := ioutil.ReadAll(newRes.Body)
		if err != nil {
			return true, err
		}
		buf := bytes.NewBuffer(newBodyBs)

		if buf.Len() != len(notChangeRespStr) && buf.String() != notChangeRespStr {
			resp.Header.Add("resp_modifier_body_len", fmt.Sprintf("%d|%d", rawBodyBf.Len(), len(newBodyBs)))

			resp.Body = ioutil.NopCloser(buf).(io.ReadCloser)
			resp.ContentLength = int64(buf.Len())
			if resp.Header.Get("Content-Length") != "" {
				resp.Header.Set("Content-Length", fmt.Sprintf("%d", buf.Len()))
			}
		}
		break
	}
	return
}

func (item *ApiRespModifier) Init() error {
	if item.Enable {
		_, err := item.genVm()
		if err != nil {
			return err
		}
	}else{
		return nil
	}
	size := 10
	item.jsRunTime = make(chan otto.Value, size)

	for idx := 0; idx < size; idx++ {
		fn, err := item.genVm()
		if err != nil {
			return err
		}
		item.jsRunTime <- *fn
	}
	return nil
}

func (item *ApiRespModifier) genVm() (fn *otto.Value, err error) {
	jsTpl := Assest.GetContent("/res/sjs/modify-response.min.js")
	jsStr := strings.Replace(jsTpl, "API_FRONT_CUSTOM_JS", item.Rule, 1)
	vm := otto.New()
	vm.Run(jsStr)
	fnv, e := vm.Get("get_modify_response_api_url")
	if e != nil {
		return &fnv,nil
	}
	fn=&fnv
	////=====================================================
	//对用户的js 代码进行一个基本的测试
	fn_test, e := vm.Get("test_get_modify_response_api_url")
	if e != nil {
		return nil, e
	}
	val, e := fn_test.Call(fn_test)
	if e != nil {
		return nil, e
	}
	if val.IsNull() {
		return
	}
	if val.IsBoolean() {
		_bv, _ := val.ToBoolean()
		if !_bv {
			return
		}
		return nil, fmt.Errorf("return value now allow [true]")
	}
	
	if val.IsString() {
		_sv, _ := val.ToString()
		_sv=strings.TrimSpace(_sv)
		switch _sv {
		case "not":
		case "":
			return
		default:
			_url, e := url.Parse(_sv)
			if e != nil {
				return nil, e
			}
			if _url.Scheme != "http" && _url.Scheme != "https" {
				return nil, fmt.Errorf("return url not allow [%s]", _sv)
			}
			return
		}
	}
	////=====================================================
	return nil,fmt.Errorf("other return value not allowed:%s",val.String())
}
