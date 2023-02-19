package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	// "net/url"

	"github.com/antonholmquist/jason"
	"golang.org/x/oauth2"
)

type oauth2Conf struct {
	Type             string                `json:"type"`
	Enable           bool                  `json:"enable"`
	ClientID         string                `json:"client_id"`
	ClientSecret     string                `json:"client_sk"`
	Scopes           []string              `json:"scopes"`
	AuthURL          string                `json:"auth_url"`
	BrokenAuthHeader bool                  `json:"broken_auth_header"` // 获取token时是否不支持header模式
	TokenURL         string                `json:"token_url"`
	Apis             map[string]*oauth2Api `json:"apis"`
	FieldMap         map[string]string     `json:"field_map"`
	config           *oauth2.Config
}

type oauth2Api struct {
	ApiUrl   string          `json:"url"`
	FieldMap oauth2ApiFields `json:"field_map"`
}

const OAUTH2_API_USER_INFO = "user_info"

type oauth2ApiFields map[string]string

var oauth2ApiFieldsDefault = map[string]map[string]string{
	OAUTH2_API_USER_INFO: {
		// 标准名字，当前名字
		"id":        "id",
		"nick_name": "name",
		"email":     "email",
		"picture":   "picture",
	},
}

func (conf *oauth2Conf) getOauthApi(name string) *oauth2Api {
	if api, has := conf.Apis[name]; has {
		return api
	}
	log.Fatalln("oauth2 api ", name, " not config")
	return nil
}

func (conf *oauth2Conf) getOauthUrl(redirectURL string) string {
	config := &oauth2.Config{
		ClientID:     conf.ClientID,
		ClientSecret: conf.ClientSecret,
		Scopes:       conf.Scopes,
		RedirectURL:  redirectURL,
		Endpoint: oauth2.Endpoint{
			AuthURL:  conf.AuthURL,
			TokenURL: conf.TokenURL,
		},
	}
	urlStr := config.AuthCodeURL("state", oauth2.AccessTypeOffline)
	// 	if(conf.FieldMap!=nil){
	// 		u,_:=url.Parse(urlStr)
	// 		qs:=u.Query()
	// 		for key,keyNow:=range conf.FieldMap{
	// 			qs[keyNow]=qs[key]
	// 			delete(qs,key)
	// 		}
	// 		log.Println("qsqs",qs)
	// 		u.RawQuery=qs.Encode()
	// 		urlStr=u.String()
	// 	}

	if conf.BrokenAuthHeader {
		oauth2.RegisterBrokenAuthHeaderProvider(conf.TokenURL)
	}

	conf.config = config
	return urlStr
}

func (conf *oauth2Conf) checkConf() {
	api := conf.getOauthApi(OAUTH2_API_USER_INFO)
	if api.FieldMap == nil {
		api.FieldMap = oauth2ApiFields{}
	}
	for k, v := range oauth2ApiFieldsDefault[OAUTH2_API_USER_INFO] {
		cv, has := api.FieldMap[k]
		if !has || cv == "" {
			api.FieldMap[k] = v
		}
	}
}

func (conf *oauth2Conf) getAccessToken(req *http.Request) (*oauth2.Token, error) {
	code := req.FormValue("code")

	tok, err := conf.config.Exchange(req.Context(), code)
	if err != nil {
		log.Println("Exchange failed,tok:", tok, "error:", err)
		return nil, err
	}
	return tok, err
}

func (conf *oauth2Conf) getUserInfo(tok *oauth2.Token) (*User, error) {
	client := conf.config.Client(context.Background(), tok)

	api := conf.getOauthApi(OAUTH2_API_USER_INFO)
	resp, err := client.Get(api.ApiUrl)
	if err != nil {
		log.Println("call api faild,error:", err)
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Println("getOauthInfo:", string(data))
	obj, err := jason.NewObjectFromBytes(data)
	if err != nil {
		return nil, err
	}
	log.Println("user_info_obj:", obj)

	vs := make(map[string]string)
	for key, keyNow := range api.FieldMap {
		val, _ := obj.GetString(keyNow)
		vs[key] = val
	}
	if vs["id"] == "" {
		return nil, fmt.Errorf("response has no user info:" + string(data))
	}
	user := &User{
		ID:       vs["id"],
		NickName: vs["nick_name"],
		Picture:  vs["picture"],
		Email:    vs["email"],
	}
	return user, nil
}
