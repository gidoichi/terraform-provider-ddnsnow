package ddnsnow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

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

func handleResponse(resp *http.Response) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	var ddnsNowResp ddnsNowResponse
	if err := json.Unmarshal(body, &ddnsNowResp); err != nil {
		return fmt.Errorf("unmarshal body: %w", err)
	}
	if ddnsNowResult(ddnsNowResp.Result) == ddnsNowResultNG {
		return fmt.Errorf("ddnsnow: code=%d, msg=%s", ddnsNowResp.ErrorCode, ddnsNowResp.ErrorMsg)
	}

	return nil
}
