
$().ready(function(){
	function my_checkbox(){
		var name=$(this).data("name")
		$(this).next("[name="+name+"]").remove();
		var val=$(this).is(":checked")?$(this).val():0;
		$(this).after("<input type='hidden' name='"+name+"' value='"+val+"'>")
	}
	$(".my-checkbox").each(my_checkbox);
	$("body").delegate(".my-checkbox","click",my_checkbox)
	
	$("#form_cookie_pref").submit(function(){
		var apiName=$(this).find("[name=api_name]").val()
		
		var hosts=[];
		$(this).find("[name=host_names]:checked").each(function(){
			hosts.push($(this).val())
		});
		
		$.get("/_pref",{name:apiName,host:hosts.join(",")},function(data){
			alert(data.msg)
			if(data.code==0){
				location.reload()
			}
		})
		return false;
	});
});

var socket = io("",{path:"/_socket.io/"});
socket.on('hello', function(msg){
	console && console.log(msg)
});


function proxy_api_host_add(){
    var tpl=$("#api_host_tpl").clone();
    var html=tpl.html().replace("tpl_api_proxy","").replace("http://127.0.0.1/","")
    $("#fieldset_hosts").append(html)
}

function proxy_api_host_delete(obj){
    var div=$(obj).parents("div.row");
    var url=div.find("input[name=host_url]").val();
    if(url!="" && !confirm("确定要删除(url="+url+")吗?")){
        return false;
    }
    div.remove()
    return false;
}

function proxy_api_get_pv(name,target){
	setInterval(function(){
		socket.emit("api_pv",name)
	},1000);
	socket.on("api_pv",function(data){
		$(target).html(""+data.pv);
	})
	
}
function h(html) {
	if(html==""){
		return "&nbsp;";
	}
	html = (html+"").replace(/&/g, '&amp;')
				.replace(/</g, '&lt;')
				.replace(/>/g, '&gt;')
			    .replace(/'/g, '&acute;')
			    .replace(/"/g, '&quot;')
	            .replace(/\|/g, '&brvbar;');
    return html;
}
