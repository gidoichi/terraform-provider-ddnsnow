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
			if _, err := w.Write([]byte(`<html><textarea id="update_data_txt">dummy</textarea></html>`)); err != nil {
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
  type  = "TXT"
  value = "dummy"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ddnsnow_record.test", "type", "TXT"),
					resource.TestCheckResourceAttr("ddnsnow_record.test", "value", "dummy"),
				),
			},
			// Update and Read testing
			{
				Config: fmt.Sprintf(providerConfigTpl, testServer.URL) + `
resource "ddnsnow_record" "test" {
  type  = "TXT"
  value = "dummy"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("ddnsnow_record.test", "type", "TXT"),
					resource.TestCheckResourceAttr("ddnsnow_record.test", "value", "dummy"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
