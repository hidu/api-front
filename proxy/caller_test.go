package proxy

import (
	"net/http"
	"testing"
)

func Test_CpfIP(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://127.0.0.1/", nil)

	ip_0 := "192.168.8.11"
	req.Header.Set("X-Real-Ip", ip_0)

	api := &Api{
		Name: "test",
	}

	cpf := NewCallerPrefConfByHttpRequest(req, api)
	if ip_0 != cpf.GetIp() {
		t.Error("ip wrong")
	}

	caller := NewCaller()

	caller.AddNewCallerItem(NewCallerItemMust(IP_ALL))
	item := caller.getCallerItemByIp(ip_0)

	if item.Ip != IP_ALL {
		t.Error("get ip failed")
	}
	caller.AddNewCallerItem(NewCallerItemMust(ip_0))

	item = caller.getCallerItemByIp(ip_0)

	if item.Ip != ip_0 {
		t.Error("get ip wrong,cur_ip:", ip_0, "get_ip:", item.Ip)
	}
}
