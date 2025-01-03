package ddnsnow

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Client interface {
	GetRecord(record Record) (Record, error)
	CreateRecord(record Record) error
	UpdateRecord(oldRecord, newRecord Record) error
	DeleteRecord(record Record) error
}

var _ Client = &client{}

type client struct {
	httpClient *http.Client
	uiURL      url.URL
	uiCookie   string
}

func NewClient(username, passwordHash, server *string) (*client, error) {
	var err error
	var uiURL *url.URL
	if *server != "" {
		uiURL, err = url.Parse(*server)
		if err != nil {
			return nil, fmt.Errorf("server URL parsing: %w", err)
		}
	} else {
		uiURL = &url.URL{
			Scheme: "https",
			Host:   "f5.si",
		}
	}
	uiURL.Path = "/control.php"

	uiCookie := fmt.Sprintf("cookie_loginuser=domain%%3D%s%%3Bpassword_hash%%3D%s%%3B", *username, *passwordHash)

	return &client{
		httpClient: &http.Client{},
		uiURL:      *uiURL,
		uiCookie:   uiCookie,
	}, nil
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

	return handleResponse(resp)
}

func (c *client) GetSettings() (*settings, error) {
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

	return parseSettings(resp.Body)
}

func (c *client) GetRecord(record Record) (Record, error) {
	return record, nil
}

func (c *client) CreateRecord(record Record) error {
	settings, err := c.GetSettings()
	if err != nil {
		return err
	}

	if err := settings.addRecord(record); err != nil {
		return err
	}

	return c.queryUI(settings.values())
}

func (c *client) UpdateRecord(oldRecord, newRecord Record) error {
	if oldRecord.Type != newRecord.Type {
		return fmt.Errorf("type mismatch: old=%s, new=%s", oldRecord.Type, newRecord.Type)
	}

	settings, err := c.GetSettings()
	if err != nil {
		return err
	}

	settings.removeRecord(oldRecord)
	if err := settings.addRecord(newRecord); err != nil {
		return err
	}

	return c.queryUI(settings.values())
}

func (c *client) DeleteRecord(record Record) error {
	settings, err := c.GetSettings()
	if err != nil {
		return err
	}

	settings.removeRecord(record)

	return c.queryUI(settings.values())
}
