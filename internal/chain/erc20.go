package chain

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
)

const erc20ABIJSON = `[
  {"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
  {"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"}
]`

type ERC20Metadata struct {
	Address     common.Address
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
}

func (c *Client) ERC20Metadata(ctx context.Context, address common.Address) (ERC20Metadata, error) {
	parsed, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		return ERC20Metadata{}, err
	}
	contract := bind.NewBoundContract(address, parsed, c.eth, c.eth, c.eth)
	opts := &bind.CallOpts{Context: ctx}

	var name []interface{}
	if err := contract.Call(opts, &name, "name"); err != nil {
		name = []interface{}{""}
	}
	var symbol []interface{}
	if err := contract.Call(opts, &symbol, "symbol"); err != nil {
		symbol = []interface{}{""}
	}
	var decimals []interface{}
	if err := contract.Call(opts, &decimals, "decimals"); err != nil {
		decimals = []interface{}{uint8(0)}
	}
	var totalSupply []interface{}
	if err := contract.Call(opts, &totalSupply, "totalSupply"); err != nil {
		totalSupply = []interface{}{big.NewInt(0)}
	}

	return ERC20Metadata{
		Address:     address,
		Name:        asString(name[0]),
		Symbol:      asString(symbol[0]),
		Decimals:    asUint8(decimals[0]),
		TotalSupply: asBigInt(totalSupply[0]),
	}, nil
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func asUint8(v interface{}) uint8 {
	n, _ := v.(uint8)
	return n
}

func asBigInt(v interface{}) *big.Int {
	n, ok := v.(*big.Int)
	if !ok || n == nil {
		return big.NewInt(0)
	}
	return n
}
