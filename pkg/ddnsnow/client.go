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
	CreateRecord(record Record) (Record, error)
	DeleteRecord(typ RecordType) error
	UpdateRecord(record Record) (Record, error)
}

var _ Client = &client{}

type client struct {
	httpClient *http.Client
	apiURL     url.URL
	apiHost    []string
	apiQuery   url.Values
	uiURL      url.URL
	uiCookie   string
}

func NewClient(username, apiToken, passwordHash *string) (*client, error) {
	apiURL := url.URL{
		Scheme: "https",
		Path:   "/update.php",
	}
	apiHost := []string{"f5", "si"}
	apiQuery := url.Values{
		"domain":   {*username},
		"password": {*apiToken},
		"format":   {"json"},
	}
	uiURL := url.URL{
		Scheme: "https",
		Host:   "f5.si",
		Path:   "/control.php",
	}
	uiCookie := fmt.Sprintf("cookie_loginuser=domain%%3D%s%%3Bpassword_hash%%3D%s%%3B", *username, *passwordHash)

	return &client{
		httpClient: &http.Client{},
		apiURL:     apiURL,
		apiHost:    apiHost,
		apiQuery:   apiQuery,
		uiURL:      uiURL,
		uiCookie:   uiCookie,
	}, nil
}

func (c *client) url(record Record) (url.URL, error) {
	constructed := c.apiURL

	constructed.Host = strings.Join(c.apiHost, ".")

	query := c.apiQuery
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
		return url.URL{}, fmt.Errorf("unsupported record type: %s", record.Type)
	}
	constructed.RawQuery = query.Encode()

	return constructed, nil
}

func (c *client) query(url url.URL) error {
	resp, err := c.httpClient.Get(url.String())
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

	var key string
	switch typ {
	case RecordTypeA:
		key = "update_data_a"
	case RecordTypeAAAA:
		key = "update_data_aaaa"
	case RecordTypeCNAME:
		key = "update_data_cname"
	case RecordTypeTXT:
		key = "update_data_txt"
	case RecordTypeNS:
		key = "update_data_ns"
	default:
		return Record{}, fmt.Errorf("unsupported record type: %s", typ)
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

func (c *client) CreateRecord(record Record) (Record, error) {
	url, err := c.url(record)
	if err != nil {
		return Record{}, err
	}

	if err := c.query(url); err != nil {
		return Record{}, err
	}

	return record, nil
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

	ukey := "UKEY@061e10718b1455b638af4a55a8377a01"
	values.Add("action", "update")
	values.Add("json", "1")
	values.Add("ukey", ukey)
	switch typ {
	case RecordTypeA:
		values.Del("update_data_a")
	case RecordTypeAAAA:
		values.Del("update_data_aaaa")
	case RecordTypeCNAME:
		values.Del("update_data_cname")
	case RecordTypeTXT:
		values.Del("update_data_txt")
	case RecordTypeNS:
		values.Del("update_data_ns")
	default:
		return fmt.Errorf("unsupported record type: %s", typ)
	}

	req, err := http.NewRequest("POST", c.uiURL.String(), strings.NewReader(values.Encode()))
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

func (c *client) UpdateRecord(record Record) (Record, error) {
	return Record{}, nil
}
