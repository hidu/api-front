package server
import (
	"github.com/hidu/api-front/models"
	"log"
	"net/http"
	"net/url"
	"time"
	"net"
	"io"
	"strings"
	"sync"
)


type APIBackend struct{
	ModelBackEnd *models.Backend
	UrlNew string
	UrlQuery url.Values
	Rw sync.RWMutex
}

func NewAPIBackendByMod(mod *models.Backend) *APIBackend{
	return &APIBackend{
		ModelBackEnd:mod,
	}
}

func NewApiBackend(id int64)(*APIBackend,error){
	b:=&models.Backend{}
	b.ID=id
	err:=b.Read()
	if err!=nil{
		return nil,err
	}
	return NewAPIBackendByMod(b),nil
}

func (a *APIBackend)ServeHTTP(rw http.ResponseWriter, req *http.Request,relPath string) {
	a.Rw.RLock()
	defer a.Rw.RUnlock()
	
	timeoutMs := time.Duration(3000) * time.Millisecond
	transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   timeoutMs,
				KeepAlive: 0,
			}).Dial,
			TLSHandshakeTimeout: timeoutMs,
			DisableKeepAlives:   true,
		}
	reqNew,err:=a.GetRealRequest(req, relPath)
	
	if err!=nil{
		rw.WriteHeader(http.StatusBadGateway)
		rw.Write([]byte(err.Error()))
		return
	}
	log.Println("call_url",reqNew.URL.String())
	resp,err:=transport.RoundTrip(reqNew)
	if err!=nil{
		rw.WriteHeader(http.StatusBadGateway)
		rw.Write([]byte(err.Error()))
		return
	}
	rw.WriteHeader(resp.StatusCode)
	defer resp.Body.Close()
	
	_, io_err := io.Copy(rw, resp.Body)
	if io_err != nil {
		log.Println("io.copy err:",io_err)
		return
	}
	
}

func (a *APIBackend)GetRealRequest(req *http.Request,relPath string)(*http.Request,error){
	urlNew:=a.UrlNew+relPath
	if req.URL.RawQuery!=""{
		urlNew+="?"+req.URL.RawQuery
	}
	r,err:=http.NewRequest(req.Method, urlNew, req.Body)
	
	if err!=nil && len(a.UrlQuery)>0{
		for k,vs:=range a.UrlQuery{
			r.URL.Query().Del(k)
			for _,v:=range vs{
				r.URL.Query().Add(k, v)
			}
		}
	}
	return r,err
}

func (a *APIBackend)Version()int64{
	return a.ModelBackEnd.UpdateTime.UnixNano()
}
func (a *APIBackend)Reload()error{
	a.Rw.Lock()
	defer a.Rw.Unlock()
	
	u,err:=url.Parse(a.ModelBackEnd.URLStr)
	if err!=nil{
		return err
	}
	a.UrlQuery=u.Query()
	tmp:=strings.Split(a.ModelBackEnd.URLStr,"?")
	a.UrlNew=tmp[0]
	//todo
	return nil
}
func (a *APIBackend)GetBackendID()int64{
	return a.ModelBackEnd.ID
}

func (a *APIBackend)Stop()error{
	//todo
	return nil
}