package ddnsnow

type ddnsNowResponse struct {
	Result    string `json:"result"`
	ErrorCode int    `json:"errorcode"`
	ErrorMsg  string `json:"errormsg"`
	RemoteIP  string `json:"remote_ip"`
}

type ddnsNowResult string

var (
	ddnsNowResultOK ddnsNowResult = "OK"
	ddnsNowResultNG ddnsNowResult = "NG"
)
