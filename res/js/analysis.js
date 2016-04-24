
var allow_receive_req=true

//每5秒ping一次,后端则>5秒去check存活
function sendServiceNotice(){
    socket.emit("http_analysis",api_front_apiName);
    setTimeout(function(){
        sendServiceNotice();
    },5000);
}

socket.on("connect",function(msg){
	console && console.log("socket.io connect",msg)
	sendServiceNotice();
	$("#connect_status").html("<font color=green>listening</font>");
})

socket.on("disconnect",function(msg){
    $("#connect_status").html("<font color=red>offline</font>");
})


socket.on("s_http_analysis",function(msg){
    console && console.log("socket.io http_analysis",msg,new Date())
    var id="api_ana_user_"+api_front_apiName;
    $("#"+id).html("<font color=blue>"+msg.client_num+"</font>")
})

socket.on("req",function(req){
	if(!allow_receive_req){
		console && console.log("receive and skiped")
		return
	}
	console && console.log("receive req data:",req)
	if(req && typeof req =="object"){
		try{
			showReqDetail(req)
		}catch(e){
			console && console.log("showReqDetail err:",e)
		}
	}
})
var req_max_length=500;
var req_list=[];
var localStrName="ap_front_reqs_"+api_front_apiName
try{
	if(window.localStorage && window.localStorage[localStrName]){
		req_list=$.parseJSON(window.localStorage[localStrName]||"[]")
	}
}catch(e){
	console&& console.log(e)
}

function req_clean(){
	req_list=[]
	$("#req_list_filter_body").empty()
}

$().ready(function(){
	for(var i=0;i<req_list.length;i++){
		addReqFilter(req_list[i])
	}
})

function addReqFilter(req){
	var uri=req.data["path"]||"";
	var _method=req.data.method
	if(_method=="GET"){
		_method="<a target='_blank' href='"+h(req.data["path"])+"' title='重新请求(cookie和其他header将丢失)' >"+req.data.method+"</a>"
	}else if (_method=="POST"){
		var form_str=buildReplayForm(req.data["req_detail"]||"",uri,"POST")
		if(form_str!==false){
			_method=form_str
		}
	}	
	var tr=$("<tr id='tr_"+h(req.id)+"'><th>"+short_id(req.id)+"</th>" +
			"<td>"+_method+"</td>" +
		    "<td title='" +h(req.data.path)+"'> <input type='text' value='" +h(req.data.path)+"' readonly>"+
			"</td><td>"+h(req.data.resp_status)+"</td></tr>");
	tr.data("req",req)
	tr.click((function(req){
		return function(){
			showReqTr(req)
			$("#req_list_filter_body tr").removeClass("success")
			tr.addClass("success")
		}
	})(req));
	filterReqsTr(tr)
	$("#req_list_filter_body").prepend(tr)
}

function showReqTr(req){
	location.hash=req.id
	var uri=req.data["path"]||"";
//	var uri_prex=$.trim($("#form_analysis_filter").find("[name=uri_prex]").val())
//	if(uri_prex!="" && uri.match(uri_prex)!=uri_prex ){
//	    console && console.log("filter,uri",uri);
//	    return;
//	}
	
	var tr="<div class='panel panel-default'>" +
			"<div class='panel-heading'>" +
			req.data.method+"&nbsp;&nbsp;"+h(req.data["path"])+
			"</div>" +
			"<div class='panel-body'>" +
			"<ul class='list-inline'>" +
			"<li>id: "+req.id+"</li>" +
			"<li>status: "+h(req.data["resp_status"]||502)+"</li>" +
			"<li>remote: "+req.data.remote+"</li>"+
			"<li>master: "+h(req.data.master)+"</li>"+
			"<li>used: "+(req.data.used && req.data.used.toFixed(2))+" ms</li>"+
			"</ul></div></div>";
	
	tr+="<div>" +
			"<pre>"+(formatReqData(req.data["req_detail"]||"",req.data["path"]||""))+"</pre>" +
			"<pre>"+
			(req.data.err?h(req.data.err||""):"")+
			showDumpData(req.data["res_detail"]||"")+"</pre>" +
			"</div>"
	$("#div_resp_detail").empty().html(tr).slideDown()
	
//	$("#req_list tr.req_tr").each(function(index,data){
//		if(index>=req_max_length){
//			data.next("tr").remove();
//			data.remove();
//		}
//	})
}

function buildReplayForm(str,uri,button_txt){
	var pos=str.indexOf("\r\n\r\n");
	var hd=str.substr(0,pos+4)+""
	var bd=str.substr(pos+4)+""
	var isForm=hd.indexOf("x-www-form-urlencoded")>0||(bd=="")
	if(!isForm){
		return false;
	}
	var form="<form method='post' action='"+uri+"' target='_blank'>"
	
	var arr=bd.split("&")
	for(var i=0;i<arr.length;i++){
		var item=arr[i].split("=")
		var k=item[0],v=urldecode(item[1]||"")
		form+="<input type='hidden' name='"+k+"' value='"+h(v)+"'>"
	}
	
	form+="<input type='submit' class='btn btn-link my-btn-post' value='"+button_txt+"' title='重放表单（cookie和其他header将丢失）'>"
	form+="</form>"
	return form
}

function formatReqData(str,path){
	str+=""
	if(str.length==0){
		return str
	}
	var pos=str.indexOf("\r\n\r\n");
	var hd=str.substr(0,pos+4)+""
	var bd=str.substr(pos+4)+""
	var result=h(str)

	var isForm=hd.indexOf("x-www-form-urlencoded")>0
	var line="<----------------------------------\n"
	var jsonBd=parseAsjson(bd)
	var pos_query=path.indexOf("?")
	if(pos_query && pos_query>0){
		var query=path.substr(pos_query+1)+""
		if(query!=""){
		    result+="<table class='table table-hover'><caption>GET Params</caption>" +
		    		"<thead><tr><th width='50px'>no</th><th>key</th><th>value</th><th>value_encode</th></tr></thead>" +
		    		"<tbody>"
			var arr=query.split("&")
			for(var i=0;i<arr.length;i++){
				var p=arr[i].split("=")
				var v=p[1]||''
				result+="<tr><td>"+(i+1)+"</td><td>"+h(p[0])+"</td><td>"+urldecode(v)+"</td><td>"+h(v)+"</td></tr>"
			}
			result+="</tbody></table>"
		}
	}
	
	if(jsonBd!=false){
		result+=jsonBd
	}else if(isForm){
		var bodyFormat=""
		bodyFormat+="<table class='table table-hover'><caption>Body Params</caption>" +
 		"<thead><tr><th width='50px'>no</th><th>key</th><th>value pretty</th><th>value_encode</th></tr></thead>" +
 		"<tbody>"
		var arr=bd.split("&")
		for(var i=0;i<arr.length;i++){
			var item=arr[i].split("=")
			var k=item[0],v=item[1]||""
			var v_raw=urldecode(v)
			var vjosn=parseAsjson(v_raw)
			var v_format=v_raw;
			if(false!=vjosn){
				v_format="<pre>"+vjosn+"</pre>"
			}
			bodyFormat+="<tr><td>"+(i+1)+"</td><td>"+h(k)+"</td><td>"+v_format+"</td><td>"+v+"</td></tr>"
		}
		bodyFormat+="</tbody></table>"
		result+=bodyFormat
	}else{
		result+=h(bd)
	}
	
	
	
	return result
}

function showDumpData(str){
	var pos=str.indexOf("\r\n\r\n");
	var hd=str.substr(0,pos+4)
	var bd=$.trim(str.substr(pos+4))
	var jsonBd=parseAsjson(bd)
	
	var result=h(hd)
	var flag_body=false
	if(jsonBd!=false){
		result+="\n<---------body---format------------------\n"+jsonBd
		flag_body=true
	}
	var header=parserHttpHeader(hd)
	var ct=header["Content-Type"]||""
	
	if(!flag_body && bd.length>10 &&ct.match(/image\//)){
		result+='<img src="data:'+ct+';base64,'+base64_encode(bd)+'">'
		flag_body=true
	}
	
	if(!flag_body){
		result+=h(bd)
		
	}
	return result
}


function parserHttpHeader(str){
	var lines=str.split("\r\n")
	var obj={}
	for(var i=0;i<lines.length;i++){
		var line=lines[i];
		var pos=line.indexOf(":")
		if(pos<1){
			continue
		}
		var k=$.trim(line.substring(0,pos))
		if(k==""){
			continue
		}
		var v=$.trim(line.substring(pos+1))
		obj[k]=v
	}
	return obj
}

function parseAsjson(str) {
	if(typeof str!="string"){
		return false
	}
	if(str.length<2){
		return false
	}
    try {
    	if(str[0]!="{" && str[0]!="["){
    		return false;
    	}
        var jsonObj = JSON.parse(str);
        if (jsonObj) {
        	jsonObj=revParseJson(jsonObj)
           return JSON.stringify(jsonObj, null, 4);
        }
    } catch (e) {
    	console.log("parseAsjson_error",e)
    }
    return false;
}

function revParseJson(obj){
	var t=typeof obj
	if(!$.isArray(obj) && t!="object"){
		return obj
	}
	var objNew=$.isArray(obj)?[]:{}
	$.each(obj,function(k,v){
		objNew[k]=revParseJson(v)
		if(typeof v=="string" && v.length>2 && (v[0]=="["||v[0]=="{")){
			try{
				var tmp=JSON.parse(v);
				if(tmp!=false){
					objNew[k+"_json_decode"]=tmp
				}
			}catch(e){
			}
		}
	})
	return objNew
}
	
function showReqDetail(req){
	if(req && req.data){
	   req.data.req_detail=base64_decode(req.data.req_detail)
	   req.data.res_detail=base64_decode(req.data.res_detail)
	}
	addReqFilter(req)
	req_list.push(req)
	while(req_max_length>0 && req_list.length>req_max_length){
		req_list.shift();
	}
}

window.onbeforeunload=function(){
	if(req_max_length>0){
		window.localStorage[localStrName]=JSON.stringify(req_list)
	}
}

function filterReqsTr(tr,uri_prex){
	var req=$(tr).data("req")
	if(!uri_prex){
		uri_prex=$("#form_analysis_filter").find("[name=uri_prex]").val()
	}
	if(!req){
		return;
	}
	var uri=req.data.path
	var show=uri.match(uri_prex)
	if(!show){
		show=uri.indexOf(uri_prex)>=0
	}
	if(show){
		$(tr).removeClass("hide")
	}else{
		$(tr).addClass("hide")
	}
}

function filterReqs(uri_prex){
	if(uri_prex==""){
		$("#req_list_filter_body tr").removeClass("hide")
		return;
	}
	$("#req_list_filter_body tr").each(function(){
		filterReqsTr($(this),uri_prex)
	});
}

$().ready(function(){
	$("#item_open_all").click(function(){
		$("#req_list tr").not(".req_tr").removeClass("hidden")
		return false;
	});
	$("#item_close_all").click(function(){
		$("#req_list tr").not(".req_tr").addClass("hidden")
		return false;
	});
	$("#item_checkbox_receive").click(function(){
		allow_receive_req=$(this).is(":checked")
	});
	
	if(location.hash!="" && location.hash.length>8){
		$("#tr_"+location.hash.substr(1)).click();
	}
	
	$("#form_analysis_filter").find("[name=uri_prex]").keyup(function(){
		filterReqs($(this).val())
	})
});