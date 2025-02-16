// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ddnsnow

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type settings struct {
	Records        Records
	EnableWildcard bool
}

func ParseSettings(r io.Reader) (*settings, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var settings settings
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

		var recordType RecordType
		switch key {
		case "update_data_wildcard":
			if _, ok := attributes["checked"]; ok {
				settings.EnableWildcard = true
			}
			continue
		case "update_data_a":
			recordType = RecordTypeA
		case "update_data_aaaa":
			recordType = RecordTypeAAAA
		case "update_data_cname":
			recordType = RecordTypeCNAME
		case "update_data_txt":
			recordType = RecordTypeTXT
		case "update_data_ns":
			recordType = RecordTypeNS
		}

		var values []string
		switch key {
		case "update_data_a", "update_data_aaaa", "update_data_cname":
			if attributes["value"] != "" {
				values = []string{attributes["value"]}
			}
		case "update_data_txt", "update_data_ns":
			if node.FirstChild != nil && node.FirstChild.Data != "" {
				values = strings.Split(node.FirstChild.Data, "\n")
			}
		}

		for _, value := range values {
			settings.Records = append(settings.Records, Record{
				Type:  recordType,
				Value: value,
			})
		}
	}

	return &settings, nil
}

func (s *settings) GetRecord(record Record) (Record, error) {
	var filter func(Record) bool
	switch record.Type {
	case RecordTypeA, RecordTypeAAAA, RecordTypeCNAME:
		filter = func(r Record) bool {
			return r.Type == record.Type
		}
	case RecordTypeNS, RecordTypeTXT:
		filter = func(r Record) bool {
			return r.Type == record.Type && r.Value == record.Value
		}
	}

	return s.Records.GetOne(filter)
}

func (s *settings) RemoveRecord(record Record) error {
	records, err := s.Records.RemoveOne(record)
	if err != nil {
		return fmt.Errorf("remove record: %w", err)
	}

	s.Records = records

	return nil
}

func (s *settings) AddRecord(record Record) error {
	switch record.Type {
	case RecordTypeA, RecordTypeAAAA, RecordTypeTXT:
		records := s.Records.Filter(func(r Record) bool {
			return r.Type == RecordTypeCNAME
		})
		if len(records) > 0 {
			return fmt.Errorf("CNAME record already exists")
		}

	case RecordTypeCNAME:
		records := s.Records.Filter(func(r Record) bool {
			return r.Type == RecordTypeA || r.Type == RecordTypeAAAA || r.Type == RecordTypeTXT
		})
		if len(records) > 0 {
			return fmt.Errorf("A/AAAA/TXT record already exists")
		}
	}

	switch record.Type {
	case RecordTypeA, RecordTypeAAAA, RecordTypeCNAME:
		exists := s.Records.Filter(func(r Record) bool {
			return r.Type == record.Type
		})
		if len(exists) != 0 {
			return fmt.Errorf("record already exists: %s", record)
		}

		s.Records = s.Records.Add(record)

	case RecordTypeNS, RecordTypeTXT:
		s.Records = s.Records.Add(record)
	}

	return nil
}

func (s *settings) URLValues() url.Values {
	values := url.Values{}

	for _, record := range s.Records {
		var key string
		switch record.Type {
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
		}

		value := values.Get(key)
		if value == "" {
			values.Set(key, record.Value)
		} else {
			value = strings.Join([]string{value, record.Value}, "\n")
			values.Set(key, value)
		}
	}

	if s.EnableWildcard {
		values.Add("update_data_wildcard", "1")
	}

	return values
}
