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
