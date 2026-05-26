package chain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	eth *ethclient.Client
}

func Dial(ctx context.Context, rpcURL string) (*Client, error) {
	eth, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	return &Client{eth: eth}, nil
}

func (c *Client) Close() {
	c.eth.Close()
}

func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	return c.eth.ChainID(ctx)
}

func (c *Client) LatestBlockNumber(ctx context.Context) (uint64, error) {
	return c.eth.BlockNumber(ctx)
}

func (c *Client) Eth() *ethclient.Client {
	return c.eth
}
