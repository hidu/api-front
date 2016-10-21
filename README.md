# api-front



### 使用源码安装

使用go get命令安装：  
```
go get -u github.com/hidu/api-front
```

## 运行
```
api-front -conf ./conf/server.json
```


## 配置

### 主配置文件
<p>conf/server.json</p>
```
{
  "users":["admin"],
  "store_api_url":"http://127.0.0.1/test/store.php",
  "store_view_url":"http://127.0.0.1/test/view.php?host_id={host_id}&api_name={api_name}"
}
```
注：admin用户有所有权限。  
store_api_url: 远程保存请求详情的地址，发送post请求（同时需要下列子服务配置中的store=true才会生效）  
store_view_url: 查看接口历史数据的页面地址  


### 子服务配置
<p>conf/vhost/8080.json</p>
```
{
    "port": 8080,
    "group": "order",
    "name": "order api",
    "enable": true,
    "hidden_cookie": true,
    "note": "订单相关接口",
    "users": [
        "test"
    ]
    "store":true
}
```
访问 http://127.0.0.1:8080/ 即可进入管理页面。  
注：test用户有当前子服务的所有权限。


### 用户配置
文件名：conf/users  ，普通文本文件：  
```
id:admin psw_md5:7bb483729b5a8e26f73e1831cde5b842 psw:psw
id:test psw_md5:7bb483729b5a8e26f73e1831cde5b842 psw:psw
```
修改服务配置需要登陆。

### 登陆控制
默认是采用普通的用户名/密码的认证方式（使用conf/users用户表）。  
目前已经支持oauth2来登陆认证，若配置oauth2信息并且enable=true则使用oauth2登陆以进行权限控制。  
`conf/server.json` 配置如下：  
```
{
"users":["xxxx"],
"oauth2":{
  "type":"google",
   "enable":true,
  "client_id":"your client id",
  "client_sk":"your client secret key",
  "scopes":["openid", "email"],
  "auth_url":"https://accounts.google.com/o/oauth2/v2/auth",
  "token_url":"https://accounts.google.com/o/oauth2/token",
   "broken_auth_header":false,
  "apis":{
      "user_info":{
        "url":"https://www.googleapis.com/oauth2/v3/userinfo",
        "field_map":{
            "id":"email"
        }
      }
  }
}
}

```
broken_auth_header：获取token时是否不支持header模式  

apis/user_info/field_map属性是用来进行属性映射的，以解决不同系统返回用户信息不一致的问题，默认值如下：
```
    //标准名字，当前名字
    "id":        "id",
    "nick_name": "name",
    "email":     "email",
    "picture":   "picture",
```

###说明
hidden_cookie:在使用协议抓包分析(analysis)是输出到前端的cookie值是否隐藏起来。  
