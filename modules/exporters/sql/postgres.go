package sql

import ipfix "github.com/CN-TU/go-ipfix"

func postgreSQLTypesTable() map[ipfix.Type]string {
	return map[ipfix.Type]string{
		ipfix.OctetArrayType:           "BYTEA",
		ipfix.Signed8Type:              "SMALLINT",
		ipfix.Unsigned8Type:            "SMALLINT",
		ipfix.Signed16Type:             "SMALLINT",
		ipfix.Unsigned16Type:           "INT",
		ipfix.Signed32Type:             "INT",
		ipfix.Unsigned32Type:           "BIGINT",
		ipfix.Signed64Type:             "BIGINT",
		ipfix.Unsigned64Type:           "BIGINT",
		ipfix.Float32Type:              "FLOAT4",
		ipfix.Float64Type:              "FLOAT8",
		ipfix.BooleanType:              "CHAR(1)",
		ipfix.MacAddressType:           "MACADDR",
		ipfix.StringType:               "TEXT",
		ipfix.DateTimeSecondsType:      "BIGINT",
		ipfix.DateTimeMillisecondsType: "BIGINT",
		ipfix.DateTimeMicrosecondsType: "BIGINT",
		ipfix.DateTimeNanosecondsType:  "BIGINT",
		ipfix.Ipv4AddressType:          "INET",
		ipfix.Ipv6AddressType:          "INET",
		ipfix.BasicListType:            "TEXT",
	}
}
