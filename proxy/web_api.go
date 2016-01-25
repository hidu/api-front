package proxy

func (wr *webReq) serverAdd() {
	wr.render("server_edit.html", true)
}
