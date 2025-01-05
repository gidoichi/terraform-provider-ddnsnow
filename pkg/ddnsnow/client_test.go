// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ddnsnow_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"terraform-provider-ddnsnow/pkg/ddnsnow"
	"testing"
)

var (
	domain       = "testdomain"
	passwordHash = "0123456789abcdef"
)

func TestClientGetSettings(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, err := w.Write([]byte(`<html>
<input type="text" id="update_data_a" value="127.0.0.1">
<textarea id="update_data_txt">record1
record2</textarea>
<input type="checkbox" id="update_data_wildcard" checked>
</html>`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
	}))
	defer testServer.Close()
	server := testServer.URL

	client, err := ddnsnow.NewClient(&domain, &passwordHash, &server)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	settings, err := client.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}

	if settings.Records[ddnsnow.RecordTypeA][0] != "127.0.0.1" {
		t.Fatalf("unexpected record value: %s", settings.Records[ddnsnow.RecordTypeA][0])
	}
	if settings.Records[ddnsnow.RecordTypeTXT][0] != "record1" {
		t.Fatalf("unexpected record value: %s", settings.Records[ddnsnow.RecordTypeTXT][0])
	}
	if settings.Records[ddnsnow.RecordTypeTXT][1] != "record2" {
		t.Fatalf("unexpected record value: %s", settings.Records[ddnsnow.RecordTypeTXT][1])
	}
	if settings.EnableWildcard != true {
		t.Fatalf("unexpected wildcard value: %t", settings.EnableWildcard)
	}
}

func TestClientGetRecord(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, err := w.Write([]byte(`<html><input type="text" id="update_data_a" value="127.0.0.1"></html>`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
	}))
	defer testServer.Close()
	server := testServer.URL

	client, err := ddnsnow.NewClient(&domain, &passwordHash, &server)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	record := ddnsnow.Record{
		Type:  ddnsnow.RecordTypeA,
		Value: "127.0.0.1",
	}
	r, err := client.GetRecord(record)
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if r != record {
		t.Fatalf("unexpected record: %v", r)
	}
}

func TestClientCreateRecord(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, err := w.Write([]byte(`<html><input type="text" id="update_data_a" value="127.0.0.1"></html>`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		case http.MethodPost:
			defer r.Body.Close()
			var body []byte
			var err error
			if body, err = io.ReadAll(r.Body); err != nil {
				t.Fatalf("ReadAll: %v", err)
			}

			values, err := url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("ParseQuery: %v", err)
			}
			if values.Get("action") != "update" ||
				values.Get("json") != "1" ||
				values.Get("update_data_a") != "127.0.0.1" ||
				values.Get("update_data_aaaa") != "::1" {
				t.Fatalf("unexpected values: %v", values)
			}

			_, err = w.Write([]byte(`{"result":"OK"}`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
	}))
	defer testServer.Close()
	server := testServer.URL

	client, err := ddnsnow.NewClient(&domain, &passwordHash, &server)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	record := ddnsnow.Record{
		Type:  ddnsnow.RecordTypeAAAA,
		Value: "::1",
	}
	if err := client.CreateRecord(record); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
}

func TestClientCreateRecordFailsWithConflictedRecordExists(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, err := w.Write([]byte(`<html><input type="text" id="update_data_a" value="127.0.0.1"></html>`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
	}))
	defer testServer.Close()
	server := testServer.URL

	client, err := ddnsnow.NewClient(&domain, &passwordHash, &server)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	record := ddnsnow.Record{
		Type:  ddnsnow.RecordTypeCNAME,
		Value: "example.com",
	}
	if err := client.CreateRecord(record); err == nil {
		t.Fatalf("CreateRecord: expected error, got nil")
	}
}

func TestClientUpdateRecord(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, err := w.Write([]byte(`<html><input type="text" id="update_data_a" value="127.0.0.1"></html>`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		case http.MethodPost:
			defer r.Body.Close()
			var body []byte
			var err error
			if body, err = io.ReadAll(r.Body); err != nil {
				t.Fatalf("ReadAll: %v", err)
			}

			values, err := url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("ParseQuery: %v", err)
			}
			if values.Get("action") != "update" ||
				values.Get("json") != "1" ||
				values.Get("update_data_a") != "127.0.0.2" {
				t.Fatalf("unexpected values: %v", values)
			}

			_, err = w.Write([]byte(`{"result":"OK"}`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
	}))
	defer testServer.Close()
	server := testServer.URL

	client, err := ddnsnow.NewClient(&domain, &passwordHash, &server)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	oldRecord := ddnsnow.Record{
		Type:  ddnsnow.RecordTypeA,
		Value: "127.0.0.1",
	}
	newRecord := ddnsnow.Record{
		Type:  ddnsnow.RecordTypeA,
		Value: "127.0.0.2",
	}
	if err := client.UpdateRecord(oldRecord, newRecord); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
}

func TestClientDeleteRecord(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, err := w.Write([]byte(`<html><input type="text" id="update_data_a" value="127.0.0.1"></html>`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		case http.MethodPost:
			defer r.Body.Close()
			var body []byte
			var err error
			if body, err = io.ReadAll(r.Body); err != nil {
				t.Fatalf("ReadAll: %v", err)
			}

			values, err := url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("ParseQuery: %v", err)
			}
			if values.Get("action") != "update" ||
				values.Get("json") != "1" ||
				values.Get("update_data_a") != "" {
				t.Fatalf("unexpected values: %v", values)
			}

			_, err = w.Write([]byte(`{"result":"OK"}`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
	}))
	defer testServer.Close()
	server := testServer.URL

	client, err := ddnsnow.NewClient(&domain, &passwordHash, &server)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	record := ddnsnow.Record{
		Type:  ddnsnow.RecordTypeA,
		Value: "127.0.0.1",
	}
	if err := client.DeleteRecord(record); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
}

func TestClientDeleteRecordFailsWhenRecordDoesNotExist(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, err := w.Write([]byte(`<html><input type="text" id="update_data_a" value="127.0.0.1"></html>`))
			if err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
	}))
	defer testServer.Close()
	server := testServer.URL

	client, err := ddnsnow.NewClient(&domain, &passwordHash, &server)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	record := ddnsnow.Record{
		Type:  ddnsnow.RecordTypeAAAA,
		Value: "::1",
	}
	if err := client.DeleteRecord(record); err == nil {
		t.Fatalf("CreateRecord: expected error, got nil")
	}
}
