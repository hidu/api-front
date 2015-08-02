

socket.emit("http_analysis",api_man_apiName)

socket.on("req",function(req){
	console && console.log(req)
	if(req && typeof req =="object"){
		showReqDetail(req)
	}
})
var req_max_length=100;
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
			"<td>"+h(req.data.path)+"</td>" +
			"<td>"+req.data.remote+"</td>"+
			"<td>"+h(req.data.master)+"</td>"+
			"</tr>";
	tr+="<tr class='hidden'><td colspan=5>" +
			"<pre>"+h(req.data.req_detail)+"</pre>" +
			"<pre>"+h(req.data.res_detail)+"</pre>" +
			"</td></tr>"
	$("#req_list").prepend(tr)
	
	$("#req_list tr.req_tr").each(function(index,data){
		if(index>=req_max_length){
			data.next("tr").remove();
			data.remove();
		}
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
	
});