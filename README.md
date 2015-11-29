# api-front

version :0.6.4

##概述
api front是HTTP API前端，可进行`请求代理转发`、`协议抓包分析`、`流量复制`。  


##安装

```
go get -u github.com/hidu/api-front
```

##运行
```
api-front -conf ./conf/server.json
```
##部署示例
<p><img src="/res/img/dispatch.png"></p>

##用途
###多人多模块开发演示
<p><code>1.一种情况：2个模块分别2个人开发，2个环境都能收到数据。</code></p>
<p><img src="/res/img/useage_0.png"></p>
<p><code>2.另一种情况：合作有交叉，下游一个人就收不到数据了。</code></p>
<p><img src="/res/img/useage_1.png"></p>
<p><code>3.一个理想的情况：合作有交叉时，下游每个人都能收到数据。</code></p>
<p><img src="/res/img/useage_2.png"></p>

##配置
###简单
每个端口配置一个apiserver:
conf/server.json
```
{
"server":[
    {"port":8080,"enable":true,"name":"8080测试","note":"","hidden_cookie":true},
    {"port":8081,"enable":true,"name":"test","note":"","hidden_cookie":true}
]
}

```

###高级
虚拟主机方式，一个端口可以配置多个apiserver
<p>conf/man.json</p>
```
{
"server_name":"api.xxx.com",
"server":[
    {"port":8080,"enable":true,"name":"8080测试","note":"","hidden_cookie":true},
    {"port":8081,"enable":true,"name":"test","note":"","hidden_cookie":true,"sub_domain":"test"}
]
}

```
<p>conf/vhost/order_8081.json</p>
```
{"port":8081,"enable":true,"name":"order api","note":"","hidden_cookie":true}
```
通过 order.api.xxx.com:8081 和 test.api.xxx.com:8081访问 8081端口上的apiserver。

###说明
hidden_cookie:在使用协议抓包分析(analysis)是输出到前端的cookie值是否隐藏起来。  
不使用域名方式的时候也可以使用vhos来配置多端口的apiserver。
