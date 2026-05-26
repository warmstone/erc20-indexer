package ethutil

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

var ZeroAddress = common.Address{}

func NormalizeAddress(addr common.Address) string {
	return strings.ToLower(addr.Hex())
}

func NormalizeAddressString(addr string) string {
	return strings.ToLower(common.HexToAddress(addr).Hex())
}

func IsZeroAddress(addr common.Address) bool {
	return ZeroAddress == addr
}
