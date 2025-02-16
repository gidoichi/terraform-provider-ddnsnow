// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package ddnsnow

type RecordType string

var (
	RecordTypeA     RecordType = "A"
	RecordTypeAAAA  RecordType = "AAAA"
	RecordTypeCNAME RecordType = "CNAME"
	RecordTypeNS    RecordType = "NS"
	RecordTypeTXT   RecordType = "TXT"
)

type Record struct {
	Type  RecordType
	Value string
}

type Records []Record

func (rs Records) Filter(filter func(Record) bool) Records {
	var filtered Records
	for _, record := range rs {
		if filter(record) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func (rs Records) GetOne(filter func(Record) bool) (Record, error) {
	records := rs.Filter(filter)
	if len(records) != 1 {
		return Record{}, fmt.Errorf("wrong filter: %v", records)
	}

	return records[0], nil
}

func (rs Records) Add(record Record) Records {
	return append(rs, record)
}

func (rs Records) RemoveOne(record Record) (Records, error) {
	var records Records
	var removed int
	for i, r := range rs {
		if r == record {
			records = append(rs[:i], rs[i+1:]...)
			removed++
			break
		}
	}

	if removed == 0 {
		return nil, fmt.Errorf("record not found: %s", record)
	}

	return records, nil
}
