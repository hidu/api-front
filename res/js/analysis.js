

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
			"<pre>"+h(req.data["req_detail"]||"")+"</pre>" +
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

function showDumpData(str){
	var pos=str.indexOf("\r\n\r\n");
	var hd=str.substr(0,pos+4)
	var bd=str.substr(pos+4)
	var jsonBd=parseAsjson(bd)
	if(jsonBd!=false){
		str+="\n\n------jsonBody---format-----\n"+jsonBd
	}
	return str
}

function parseAsjson(str) {
    try {
    	str=str+""
    	if(str[0]!="{" && str[0]!="["){
    		return false;
    	}
        var jsonObj = JSON.parse(str);
        if (jsonObj) {
           return JSON.stringify(jsonObj, null, 2);
        }
    } catch (e) {
    	console.log("parseAsjson_error",e)
    }
    return false;
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