
$().ready(function(){
	function my_checkbox(){
		var name=$(this).data("name")
		$(this).next("[name="+name+"]").remove();
		var val=$(this).is(":checked")?$(this).val():0;
		$(this).after("<input type='hidden' name='"+name+"' value='"+val+"'>")
	}
	$(".my-checkbox").each(my_checkbox);
	$("body").delegate(".my-checkbox","click",my_checkbox)
});