package ddnsnow

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type Client interface {
	GetRecord(typ RecordType) (Record, error)
	DeleteRecord(typ RecordType) error
	UpdateRecord(record Record) error
}

var _ Client = &client{}

type client struct {
	httpClient   *http.Client
	username     string
	apiToken     string
	passwordHash string
	apiURL       url.URL
	uiURL        url.URL
	uiCookie     string
}

func NewClient(username, apiToken, passwordHash *string) (*client, error) {
	apiURL := url.URL{
		Scheme: "https",
		Host:   "f5.si",
		Path:   "/update.php",
	}
	uiURL := url.URL{
		Scheme: "https",
		Host:   "f5.si",
		Path:   "/control.php",
	}
	uiCookie := fmt.Sprintf("cookie_loginuser=domain%%3D%s%%3Bpassword_hash%%3D%s%%3B", *username, *passwordHash)

	return &client{
		httpClient:   &http.Client{},
		username:     *username,
		apiToken:     *apiToken,
		passwordHash: *passwordHash,
		apiURL:       apiURL,
		uiURL:        uiURL,
		uiCookie:     uiCookie,
	}, nil
}

func (c *client) queryAPI(record Record) error {
	constructedURL := c.apiURL

	query := url.Values{
		"domain":   {c.username},
		"password": {c.apiToken},
		"format":   {"json"},
	}
	switch record.Type {
	case RecordTypeA:
		query.Add("ip", record.Value)
	case RecordTypeAAAA:
		query.Add("ipv6", record.Value)
	case RecordTypeCNAME:
		query.Add("cname", record.Value)
	case RecordTypeTXT:
		query.Add("txt", record.Value)
	default:
		return fmt.Errorf("unsupported record type: %s", record.Type)
	}
	constructedURL.RawQuery = query.Encode()

	resp, err := c.httpClient.Get(constructedURL.String())
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}

	return c.handleResponse(resp)
}

func (c *client) queryUI(body url.Values) error {
	ukey := "UKEY@061e10718b1455b638af4a55a8377a01"

	body.Add("action", "update")
	body.Add("json", "1")
	body.Add("ukey", ukey)

	req, err := http.NewRequest("POST", c.uiURL.String(), strings.NewReader(body.Encode()))
	if err != nil {
		return fmt.Errorf("http request construction: %w", err)
	}

	req.Header.Set("Cookie", c.uiCookie)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}

	return c.handleResponse(resp)
}

func (c *client) handleResponse(resp *http.Response) error {
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

func (c *client) setting(typ RecordType) (string, error) {
	switch typ {
	case RecordTypeA:
		return "update_data_a", nil
	case RecordTypeAAAA:
		return "update_data_aaaa", nil
	case RecordTypeCNAME:
		return "update_data_cname", nil
	case RecordTypeTXT:
		return "update_data_txt", nil
	case RecordTypeNS:
		return "update_data_ns", nil
	default:
		return "", fmt.Errorf("unsupported record type: %s", typ)
	}
}

func (c *client) GetAll() (map[string]string, error) {
	req, err := http.NewRequest("GET", c.uiURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("http request construction: %w", err)
	}

	req.Header.Set("Cookie", c.uiCookie)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	settings := map[string]string{}
	for node := range doc.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		if node.Data != "input" && node.Data != "textarea" {
			continue
		}

		attributes := map[string]string{}
		for _, attr := range node.Attr {
			attributes[attr.Key] = attr.Val
		}
		key, ok := attributes["id"]
		if !ok {
			continue
		}

		switch node.Data {
		case "input":
			if attributes["values"] != "" {
				continue
			}
			settings[key] = attributes["value"]
		case "textarea":
			if node.FirstChild == nil {
				continue
			}
			settings[key] = node.FirstChild.Data
		}
	}

	return settings, nil
}

func (c *client) GetRecord(typ RecordType) (Record, error) {
	settings, err := c.GetAll()
	if err != nil {
		return Record{}, err
	}

	key, err := c.setting(typ)
	if err != nil {
		return Record{}, err
	}

	value, ok := settings[key]
	if !ok {
		return Record{}, fmt.Errorf("record not found: %s", typ)
	}

	return Record{
		Type:  typ,
		Value: value,
	}, nil
}

func (c *client) UpdateRecord(record Record) error {
	return c.queryAPI(record)
}

func (c *client) DeleteRecord(typ RecordType) error {
	settings, err := c.GetAll()
	if err != nil {
		return err
	}

	values := url.Values{}
	for key, value := range settings {
		values.Add(key, value)
	}

	setting, err := c.setting(typ)
	if err != nil {
		return err
	}
	values.Del(setting)

	return c.queryUI(values)
}
