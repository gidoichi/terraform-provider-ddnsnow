// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ddnsnow

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type settings struct {
	Records        Records
	EnableWildcard bool
	validators     []recordsValidator
}

type recordsValidator func(Records) error

var (
	violationMultipleRecordsForSomeTypes recordsValidator = func(records Records) error {
		typeToRecords := map[RecordType][]Record{}

		for _, record := range records {
			typeToRecords[record.Type] = append(typeToRecords[record.Type], record)
		}

		var errs error
		for recordType, records := range typeToRecords {
			switch recordType {
			case RecordTypeA, RecordTypeAAAA, RecordTypeCNAME:
				if len(records) >= 2 {
					errs = errors.Join(errs, fmt.Errorf("multiple records of type %s", recordType))
				}
			}
		}

		return errs
	}

	violationConflictedRecords recordsValidator = func(records Records) error {
		cname := records.Filter(func(r Record) bool {
			return r.Type == RecordTypeCNAME
		})
		if len(cname) == 0 {
			return nil
		}

		conflicted := records.Filter(func(r Record) bool {
			return r.Type == RecordTypeA || r.Type == RecordTypeAAAA || r.Type == RecordTypeTXT
		})
		if len(conflicted) == 0 {
			return nil
		}

		return fmt.Errorf("conflicted: %v", append(conflicted, cname...))
	}
)

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

	settings.validators = []recordsValidator{
		violationMultipleRecordsForSomeTypes,
		violationConflictedRecords,
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
	updated, err := s.Records.RemoveOne(record)
	if err != nil {
		return fmt.Errorf("remove record: %w", err)
	}
	if err := s.validateRecords(updated); err != nil {
		return fmt.Errorf("remove record: %w", err)
	}

	s.Records = updated

	return nil
}

func (s *settings) AddRecord(record Record) error {
	updated := s.Records.Add(record)
	if err := s.validateRecords(updated); err != nil {
		return fmt.Errorf("add record: %w", err)
	}

	s.Records = updated

	return nil
}

func (s settings) validateRecords(records Records) error {
	var errs error
	for _, valid := range s.validators {
		if err := valid(records); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
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
