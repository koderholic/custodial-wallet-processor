package utility

var (
	AddressTypesPerAsset = map[int64][]string{
		0:   []string{ADDRESS_TYPE_SEGWIT, ADDRESS_TYPE_LEGACY},
		145: []string{ADDRESS_TYPE_LEGACY, ADDRESS_TYPE_QADDRESS},
	}
)
