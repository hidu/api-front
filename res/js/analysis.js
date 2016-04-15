
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
	$("#connect_status").html("<font color=green>online</font>");
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
	$("#req_list").empty()
}

$().ready(function(){
	for(var i=0;i<req_list.length;i++){
		showReqTr(req_list[i])
	}
})

function showReqTr(req){
	var uri=req.data["path"]||"";
	var uri_prex=$.trim($("#form_analysis_filter").find("[name=uri_prex]").val())
	if(uri_prex!="" && uri.match(uri_prex)!=uri_prex ){
	    console && console.log("filter,uri",uri);
	    return;
	}
	var _method=req.data.method
	if(_method=="GET"){
		_method="<a target='_blank' href='"+h(req.data["path"])+"' title='重新请求(cookie和其他header将丢失)' >"+req.data.method+"</a>"
	}else if (_method=="POST"){
		var form_str=buildReplayForm(req.data["req_detail"]||"",uri,"POST")
		if(form_str!==false){
			_method=form_str
		}
	}	
	
	var tr="<tr class='req_tr' data-reqid='"+req.id+"'>" +
			"<td>"+req.id+"</td>" +
			"<td>"+_method+"</td>" +
			"<td><input type='text' readonly class='td_url_input' value='"+h(req.data["path"])+"' title='"+h(req.data["path"])+"'></td>" +
			"<td>"+h(req.data["resp_status"]||502)+"</td>" +
			"<td>"+req.data.remote+"</td>"+
			"<td>"+h(req.data.master)+"</td>"+
			"<td title='ms'>"+(req.data.used && req.data.used.toFixed(2))+"</td>"
			"</tr>";
	
	tr+="<tr class='hidden'><td colspan=7>" +
			"<pre>"+(formatReqData(req.data["req_detail"]||"",req.data["path"]||""))+"</pre>" +
			"<pre>"+
			(req.data.err?h(req.data.err||""):"")+
			h(showDumpData(req.data["res_detail"]||""))+"</pre>" +
			"</td>" +
			"</tr>"
	$("#req_list").prepend(tr)
	
	$("#req_list tr.req_tr").each(function(index,data){
		if(index>=req_max_length){
			data.next("tr").remove();
			data.remove();
		}
	})
}

function buildReplayForm(str,uri,button_txt){
	var pos=str.indexOf("\r\n\r\n");
	var hd=str.substr(0,pos+4)+""
	var bd=str.substr(pos+4)+""
	var isForm=hd.indexOf("x-www-form-urlencoded")>0
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
	var result=str

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
	
	var bodyFormat=""
	if(jsonBd!=false){
		bodyFormat=jsonBd
	}else if(isForm){
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
	}
	if(bodyFormat.length>0){
		result+=bodyFormat
	}
	
	
	return result
}

function showDumpData(str){
	var pos=str.indexOf("\r\n\r\n");
	var hd=str.substr(0,pos+4)
	var bd=str.substr(pos+4)
	var jsonBd=parseAsjson(bd)
	if(jsonBd!=false){
		str+="\n<---------body---format------------------\n"+jsonBd
	}
	return str
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
	showReqTr(req)
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

$().ready(function(){
	$("#req_list").on("click","tr.req_tr",function(){
		$(this).next("tr").toggleClass("hidden");
		location.hash=$(this).data("reqid")+""
	})
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
		$("#req_list tr.req_tr").each(function(){
			if("#"+$(this).data("reqid")==location.hash){
				$(this).next("tr").toggleClass("hidden");
			}
		});
	}
});