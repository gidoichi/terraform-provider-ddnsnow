package ddnsnow

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type settings struct {
	Records        map[RecordType][]string
	EnableWildcard bool
}

func parseSettings(r io.Reader) (*settings, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	settings := settings{
		Records: map[RecordType][]string{},
	}
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

		switch key {
		case "update_data_a", "update_data_aaaa", "update_data_cname":
			if attributes["value"] == "" {
				settings.Records[recordType] = []string{}
			} else {
				settings.Records[recordType] = []string{attributes["value"]}
			}
		case "update_data_txt", "update_data_ns":
			if node.FirstChild == nil || node.FirstChild.Data == "" {
				settings.Records[recordType] = []string{}
			} else {
				settings.Records[recordType] = strings.Split(node.FirstChild.Data, "\n")
			}
		}
	}

	return &settings, nil
}

func (s *settings) getRecord(record Record) (Record, error) {
	records := s.Records[record.Type]

	switch record.Type {
	case RecordTypeA, RecordTypeAAAA, RecordTypeCNAME:
		if len(records) == 0 {
			return Record{}, fmt.Errorf("record not found: %s", record.Type)
		}
		return Record{
			Type:  record.Type,
			Value: s.Records[record.Type][0],
		}, nil

	case RecordTypeNS, RecordTypeTXT:
		records := s.Records[record.Type]
		for _, r := range records {
			if r == record.Value {
				return record, nil
			}
		}
		return Record{}, fmt.Errorf("record not found: %s", record)

	default:
		return Record{}, fmt.Errorf("unsupported record type: %s", record.Type)
	}
}

func (s *settings) removeRecord(record Record) {
	switch record.Type {
	case RecordTypeA, RecordTypeAAAA, RecordTypeCNAME:
		delete(s.Records, record.Type)

	case RecordTypeNS, RecordTypeTXT:
		records := s.Records[record.Type]
		for i, r := range records {
			if r == record.Value {
				records = append(records[:i], records[i+1:]...)
				break
			}
		}
		s.Records[record.Type] = records
	}
}

func (s *settings) addRecord(record Record) error {
	switch record.Type {
	case RecordTypeA, RecordTypeAAAA, RecordTypeCNAME:
		if len(s.Records[record.Type]) == 0 {
			s.Records[record.Type] = []string{record.Value}
		} else {
			return fmt.Errorf("record already exists: %s", record)
		}

	case RecordTypeNS, RecordTypeTXT:
		records := s.Records[record.Type]
		records = append(records, record.Value)
		s.Records[record.Type] = records
	}

	return nil
}

func (s *settings) values() url.Values {
	values := url.Values{}
	for typ, records := range s.Records {
		if len(records) == 0 {
			continue
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
		}

		values.Add(key, strings.Join(records, "\n"))
	}

	if s.EnableWildcard {
		values.Add("update_data_wildcard", "1")
	}

	return values
}
