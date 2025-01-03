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
