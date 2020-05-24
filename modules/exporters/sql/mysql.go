package sql

import ipfix "github.com/CN-TU/go-ipfix"

func mySQLTypesTable() map[ipfix.Type]string {
	return map[ipfix.Type]string{
		ipfix.OctetArrayType:           "BLOB",
		ipfix.Signed8Type:              "TINYINT",
		ipfix.Unsigned8Type:            "SMALLINT",
		ipfix.Signed16Type:             "SMALLINT",
		ipfix.Unsigned16Type:           "INT",
		ipfix.Signed32Type:             "INT",
		ipfix.Unsigned32Type:           "BIGINT",
		ipfix.Signed64Type:             "BIGINT",
		ipfix.Unsigned64Type:           "BIGINT",
		ipfix.Float32Type:              "FLOAT",
		ipfix.Float64Type:              "DOUBLE",
		ipfix.BooleanType:              "CHAR(1)",
		ipfix.MacAddressType:           "CHAR(17)",
		ipfix.StringType:               "TEXT",
		ipfix.DateTimeSecondsType:      "BIGINT",
		ipfix.DateTimeMillisecondsType: "BIGINT",
		ipfix.DateTimeMicrosecondsType: "BIGINT",
		ipfix.DateTimeNanosecondsType:  "BIGINT",
		ipfix.Ipv4AddressType:          "VARCHAR(39)",
		ipfix.Ipv6AddressType:          "VARCHAR(39)",
		ipfix.BasicListType:            "TEXT",
	}
}
