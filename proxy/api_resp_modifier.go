package proxy

import (
	"net/http"
	"github.com/robertkrimen/otto"
	"strings"
	"fmt"
	"net/url"
	"encoding/json"
	"io/ioutil"
	"bytes"
	"io"
)


type ApiRespModifier struct {
	Note   string `json:"note"`
	Enable bool   `json:"enable"`
	Rule   string `json:"rule"`
	jsRunTime chan otto.Value
}

type RespModifier []*ApiRespModifier

func newRespModifierSlice() RespModifier {
	return make([]*ApiRespModifier, 0)
}

func (rm RespModifier)Init()(error){
	for _,item:=range rm{
		err:=item.Init()
		if(err!=nil){
			return err
		}
	}
	return nil;
}
//ModifierResp 使用动态规则对response进行处理
func (rm RespModifier)ModifierResp(req *http.Request,resp *http.Response)(mod bool,err error){
	if(len(rm)==0){
		return;
	}
	items:=newRespModifierSlice()
	for _,item:=range rm{
		if(!item.Enable||item.Rule==""){
			continue;
		}
		items=append(items,item)
	}
	if(len(items)==0){
		return;
	}
	
	reqMap:=make(map[string]interface{})
	reqMap["get"]=req.URL.Query()
	reqMap["method"]=req.Method
	reqMap["url"]=req.URL.String()
	reqMap["request_uri"]=req.RequestURI
	reqMap["remote_addr"]=req.RemoteAddr
	
	_header:=make(map[string]string)
	for k,vs:=range req.Header{
		_header[k]=strings.Join(vs,",")
	}
	reqMap["remote_addr"]=_header;
	
	reqJsObj, _ := otto.New().Object(`req={}`)
	for k,v:=range reqMap{
		reqJsObj.Set(k,v);
	}
	/////////////////////////////////////////////////
	resJsObj, _ := otto.New().Object(`res={}`)
	
	resMap:=make(map[string]interface{})
	resMap["http_code"]=resp.StatusCode
	resMap["content_length"]=resp.ContentLength
	
	_res_header:=make(map[string]string)
	for k,vs:=range resp.Header{
		_res_header[k]=strings.Join(vs,",")
	}
	resMap["header"]=_res_header
	for k,v:=range resMap{
		resJsObj.Set(k,v)
	}
	//==============================================
	modReqBody:=url.Values{}
	_bs,_:=json.Marshal(reqMap)
	modReqBody.Add("req",string(_bs))
	_bs_res,_:=json.Marshal(resMap)
	modReqBody.Add("res",string(_bs_res))
	//==============================================
	hasReadBody:=false;
	notChangeRespStr:="NOT_CHANGE"
	var rawBodyBf *bytes.Buffer
	for _,item:=range items{
		fn:=<-item.jsRunTime
		js_ret, err := fn.Call(fn, reqJsObj,resJsObj)
		item.jsRunTime<-fn
		fmt.Println("js result",js_ret,err)
		if(err!=nil){
			return true,err;
		}
		if(!js_ret.IsString()){
			continue
		}
		ret_str:=js_ret.String()
		if(ret_str==""){
			continue
		}
		if(ret_str=="break"){
			break;
		}
		
		resp.Header.Set("resp_modifier",ret_str)
		
		_,err=url.Parse(ret_str)
		if(err!=nil){
			return true,err
		}
		if !hasReadBody{
			rawBodyBf=forgetRead(&resp.Body)
			hasReadBody=true
			modReqBody.Add("res_content",rawBodyBf.String())
		}
	
		newRes,err:=http.PostForm(ret_str,modReqBody)
		if(err!=nil){
			return true,err
		}
		defer newRes.Body.Close()
		newBodyBs,err:=ioutil.ReadAll(newRes.Body)
		if(err!=nil){
			return true,err
		}
		buf := bytes.NewBuffer(newBodyBs)
		
		if(buf.Len()!=len(notChangeRespStr) && buf.String()!=notChangeRespStr){
			resp.Header.Add("resp_modifier_body_len",fmt.Sprintf("%d|%d",rawBodyBf.Len(),len(newBodyBs)))
			
			resp.Body=ioutil.NopCloser(buf).(io.ReadCloser)
			resp.ContentLength=int64(buf.Len())
			if(resp.Header.Get("Content-Length")!=""){
				resp.Header.Set("Content-Length",fmt.Sprintf("%d",buf.Len()))
			}
		}
		break
	}
	return
}



func (item *ApiRespModifier)Init()(error){
	if(item.Enable){
		_,err:=item.genVm()
		if(err!=nil){
			return err;
		}
	}
	size:=10;
	item.jsRunTime=make(chan otto.Value,size)
	
	for idx:=0;idx<size;idx++{
		fn,err:=item.genVm()
		if(err!=nil){
			return err;
		}
		item.jsRunTime<-fn
	}
	return nil
}

func (item *ApiRespModifier)genVm()(fn otto.Value,err error){
	jsTpl:= Assest.GetContent("/res/sjs/resp.js")
	jsStr := strings.Replace(jsTpl, "API_FRONT_CUSTOM_JS", item.Rule, 1)
	vm:=otto.New()
	vm.Run(jsStr)
	return vm.Get("api_front_resp_rewrite")
}

