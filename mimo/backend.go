package mimo

type Backend struct {
	UrlStr string `json:url`
	Master bool   `json:"master"`
}
