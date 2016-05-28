package proxy

type ApiRespModifier struct {
	Note   string `json:"note"`
	Enable bool   `json:"enable"`
	Rule   string `json:"rule"`
}

type RespModifier []*ApiRespModifier

func newRespModifierSlice() RespModifier {
	return make([]*ApiRespModifier, 0)
}
