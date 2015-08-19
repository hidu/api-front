

socket.on("connect",function(msg){
	console && console.log("socket.io connect",msg)
	socket.emit("http_analysis",api_man_apiName)
})

socket.on("req",function(req){
	console && console.log(req)
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
var localStrName="reqs_"+api_man_apiName
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

for(var i=0;i<req_list.length;i++){
	showReqTr(req_list[i])
}

function showReqTr(req){
	
	var tr="<tr class='req_tr'>" +
			"<td>"+req.id+"</td>" +
			"<td>"+req.data.method+"</td>" +
			"<td>"+h(req.data["path"]||"unknow")+"</td>" +
			"<td>"+h(req.data["resp_status"]||502)+"</td>" +
			"<td>"+req.data.remote+"</td>"+
			"<td>"+h(req.data.master)+"</td>"+
			"<td title='ms'>"+req.data.used.toFixed(2)+"</td>"
			"</tr>";
	tr+="<tr class='hidden'><td colspan=7>" +
			"<pre>"+h(formatReqData(req.data["req_detail"]||""))+"</pre>" +
			"<pre>"+h(showDumpData(req.data["res_detail"]||""))+"</pre>" +
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

function formatReqData(str){
	str+=""
	if(str.length==0){
		return str
	}
	var pos=str.indexOf("\r\n\r\n");
	var hd=str.substr(0,pos+4)+""
	var bd=str.substr(pos+4)+""
	var result=str

	var isForm=hd.indexOf("x-www-form-urlencoded")>0
	
	var jsonBd=parseAsjson(bd)
	var bodyFormat=""
	if(jsonBd!=false){
		bodyFormat=jsonBd
	}else if(isForm){
		var arr=bd.split("&")
		for(var i=0;i<arr.length;i++){
			var item=arr[i].split("=")
			var k=item[0],v=urldecode(item[1]||"")
			bodyFormat+=(i+1)+" ) "+k+" : "+v+"\n";
			var vjosn=parseAsjson(v)
			if(false!=vjosn){
				bodyFormat+="    "+k+"_json_indent : \n"+vjosn+"\n<----------------------------------\n";
			}
		}
	}
	if(bodyFormat.length>0){
		result+="\n\n------body---format-----\n"
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
		str+="\n\n------body---format-----\n"+jsonBd
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
        	revParseJson(jsonObj)
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
	$.each(obj,function(k,v){
		if(typeof v=="string" && v.length>2 && (v[0]=="["||v[0]=="{")){
			try{
				var tmp=JSON.parse(v);
				if(tmp!=false){
					obj[k+"_json_decode"]=tmp
				}
			}catch(e){
				
			}
		}
		obj[k]=revParseJson(v)
	})
}
	
function showReqDetail(req){
	showReqTr(req)
	req_list.push(req)
	while(req_max_length>0 && req_list.length>req_max_length){ b
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
	})
	$("#item_open_all").click(function(){
		$("#req_list tr").not(".req_tr").removeClass("hidden")
		return false;
	});
	$("#item_close_all").click(function(){
		$("#req_list tr").not(".req_tr").addClass("hidden")
		return false;
	});
	
});