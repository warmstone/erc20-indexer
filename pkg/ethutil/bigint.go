package ethutil

import (
	"fmt"
	"math/big"
)

func BigIntString(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}

func MustBigInt(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(fmt.Sprintf("invalid big integer string: %q", s))
	}
	return v
}
