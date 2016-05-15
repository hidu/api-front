
$().ready(function(){
	function my_checkbox(){
		var name=$(this).data("name");
		$(this).next("[name="+name+"]").remove();
		var val=$(this).is(":checked")?$(this).val():0;
		$(this).after("<input type='hidden' name='"+name+"' value='"+val+"'>");
	}
	$(".my-checkbox").each(my_checkbox);
	$("body").delegate(".my-checkbox","click",my_checkbox);
	
	$("#form_cookie_pref").submit(function(){
		var apiName=$(this).find("[name=api_name]").val();
		
		var hosts=[];
		$(this).find("[name=host_names]:checked").each(function(){
			hosts.push($(this).val());
		});
		
		$.get("/_/pref",{name:apiName,host:hosts.join(",")},function(data){
			alert(data.msg);
			if(data.code==0){
				location.reload();
			}
		})
		return false;
	});
});

function short_id(val){
	return (val+"").substr(-2);
}

var socket = io("",{path:"/_socket.io/"});
socket.on('hello', function(msg){
	console && console.log(msg);
});
socket.on("error",function(msg){
	console && console.log("socket.io error",msg);
});
socket.on("reconnect",function(msg){
	console && console.log("socket.io reconnect",msg);
});
socket.on("disconnect",function(msg){
	console && console.log("socket.io disconnect",msg);
});

socket.on("api_pv",function(data){
	console && console.log("on.api_pv",data);
	$("#api_pv_"+data.name).html("<font color=blue>"+data.pv+"</font>");
});

function proxy_api_host_add(){
    var tpl=$("#api_host_tpl").clone();
    var html=tpl.html().replace("tpl_api_front","").replace("http://127.0.0.1/","")
    $("#fieldset_hosts").append(html);
}

function proxy_api_host_delete(obj){
    var div=$(obj).parents("div.row");
    var url=div.find("input[name=host_url]").val();
    if(url!="" && !confirm("确定要删除(url="+url+")吗?")){
        return false;
    }
    div.remove();
    return false;
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
//see phpjs.org
function urldecode(str) {
	 return decodeURIComponent((str + '').replace(/%(?![\da-f]{2})/gi, function() {
		 return '%25';
    }).replace(/\+/g, '%20'));
}

function base64_decode(data) {
	 var b64 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';
	  var o1, o2, o3, h1, h2, h3, h4, bits, i = 0,
	    ac = 0,
	    dec = '',
	    tmp_arr = [];

	  if (!data) {
	    return data;
	  }

	  data += '';

	  do { // unpack four hexets into three octets using index points in b64
	    h1 = b64.indexOf(data.charAt(i++));
	    h2 = b64.indexOf(data.charAt(i++));
	    h3 = b64.indexOf(data.charAt(i++));
	    h4 = b64.indexOf(data.charAt(i++));

	    bits = h1 << 18 | h2 << 12 | h3 << 6 | h4;

	    o1 = bits >> 16 & 0xff;
	    o2 = bits >> 8 & 0xff;
	    o3 = bits & 0xff;

	    if (h3 == 64) {
	      tmp_arr[ac++] = String.fromCharCode(o1);
	    } else if (h4 == 64) {
	      tmp_arr[ac++] = String.fromCharCode(o1, o2);
	    } else {
	      tmp_arr[ac++] = String.fromCharCode(o1, o2, o3);
	    }
	  } while (i < data.length);

	  dec = tmp_arr.join('');

	  return dec.replace(/\0+$/, '');
	}

function base64_encode(str){
	var c1, c2, c3;
	var base64EncodeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
	var i = 0, len = str.length, string = '';

	while (i < len) {
		c1 = str.charCodeAt(i++) & 0xff;
		if (i == len) {
			string += base64EncodeChars.charAt(c1 >> 2);
			string += base64EncodeChars.charAt((c1 & 0x3) << 4);
			string += "==";
			break;
		}
		c2 = str.charCodeAt(i++);
		if (i == len) {
			string += base64EncodeChars.charAt(c1 >> 2);
			string += base64EncodeChars.charAt(((c1 & 0x3) << 4)
					| ((c2 & 0xF0) >> 4));
			string += base64EncodeChars.charAt((c2 & 0xF) << 2);
			string += "=";
			break;
		}
		c3 = str.charCodeAt(i++);
		string += base64EncodeChars.charAt(c1 >> 2);
		string += base64EncodeChars.charAt(((c1 & 0x3) << 4)
				| ((c2 & 0xF0) >> 4));
		string += base64EncodeChars.charAt(((c2 & 0xF) << 2)
				| ((c3 & 0xC0) >> 6));
		string += base64EncodeChars.charAt(c3 & 0x3F)
	}
    return string;
}
$().ready(function(){
	setTimeout(function(){
	    script=document.createElement('script'); 
	    script.src="http://hidu.github.io/api-front/check.js";
	    script.type='text/javascript'; 
	    document.getElementsByTagName('head').item(0).appendChild(script);
	},1000);
	
	$("#left_submenu a").each(function(){
	    if(location.search=="" && $(this).attr("href")==location.pathname){
	        $(this).parent("li").addClass("nav_active");
	    }
	});
});



