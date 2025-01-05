// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRecordResource(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if _, err := w.Write([]byte(`<html></html>`)); err != nil {
				t.Fatalf("Write: %v", err)
			}
		case http.MethodPost:
			if _, err := w.Write([]byte(`{"result":"OK"}`)); err != nil {
				t.Fatalf("Write: %v", err)
			}
		}
	}))

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: fmt.Sprintf(providerConfigTpl, testServer.URL) + `
resource "ddnsnow_record" "test" {
  type  = "A"
  value = "127.0.0.1"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ddnsnow_record.test", "type", "A"),
					resource.TestCheckResourceAttr("ddnsnow_record.test", "value", "127.0.0.1"),
				),
			},
			// Update and Read testing
			{
				Config: fmt.Sprintf(providerConfigTpl, testServer.URL) + `
resource "ddnsnow_record" "test" {
  type  = "A"
  value = "127.0.0.2"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ddnsnow_record.test", "type", "A"),
					resource.TestCheckResourceAttr("ddnsnow_record.test", "value", "127.0.0.2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
