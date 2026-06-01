// Package node is a minimal Teranode JSON-RPC client for the regtest stack (SYS-NODE-002).
// Interface names per spec/VERIFY-LOG.md (B3/B4). Numbers marshal as JSON numbers, strings as
// JSON strings (so 64-char hex txids are never mis-read as numbers — cf. quickstart rpc.sh).
package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	URL, User, Pass string
	HTTP            *http.Client
}

func New(url, user, pass string) *Client {
	return &Client{URL: url, User: user, Pass: pass, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

type rpcReq struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}
type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
type rpcResp struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcErr         `json:"error"`
}

// Call performs a JSON-RPC 1.0 call and returns the raw result.
func (c *Client) Call(method string, params ...any) (json.RawMessage, error) {
	if params == nil {
		params = []any{}
	}
	body, _ := json.Marshal(rpcReq{JSONRPC: "1.0", ID: "te", Method: method, Params: params})
	req, err := http.NewRequest(http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.User, c.Pass)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out rpcResp
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("%s: decode (http %d): %w", method, resp.StatusCode, err)
	}
	if out.Error != nil {
		return nil, fmt.Errorf("%s: rpc error %d: %s", method, out.Error.Code, out.Error.Message)
	}
	return out.Result, nil
}

func (c *Client) strings(method string, params ...any) ([]string, error) {
	raw, err := c.Call(method, params...)
	if err != nil {
		return nil, err
	}
	var s []string
	return s, json.Unmarshal(raw, &s)
}

// Generate mines n blocks (regtest), returning the block hashes.
func (c *Client) Generate(n int) ([]string, error) { return c.strings("generate", n) }

// GenerateToAddress mines n blocks paying the coinbase to addr (regtest).
func (c *Client) GenerateToAddress(n int, addr string) ([]string, error) {
	return c.strings("generatetoaddress", n, addr)
}

type Block struct {
	Hash       string `json:"hash"`
	Height     int    `json:"height"`
	MerkleRoot string `json:"merkleroot"`
	NumTx      int    `json:"num_tx"`
}

func (c *Client) GetBlock(hash string) (Block, error) {
	var b Block
	raw, err := c.Call("getblock", hash)
	if err != nil {
		return b, err
	}
	return b, json.Unmarshal(raw, &b)
}

func (c *Client) GetBlockByHeight(h int) (Block, error) {
	var b Block
	raw, err := c.Call("getblockbyheight", h)
	if err != nil {
		return b, err
	}
	return b, json.Unmarshal(raw, &b)
}

type ChainInfo struct {
	Chain         string `json:"chain"`
	Blocks        int    `json:"blocks"`
	BestBlockHash string `json:"bestblockhash"`
}

func (c *Client) GetBlockchainInfo() (ChainInfo, error) {
	var ci ChainInfo
	raw, err := c.Call("getblockchaininfo")
	if err != nil {
		return ci, err
	}
	return ci, json.Unmarshal(raw, &ci)
}

// GetRawTransaction returns the raw transaction hex (verbose=false).
func (c *Client) GetRawTransaction(txid string) (string, error) {
	raw, err := c.Call("getrawtransaction", txid)
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}
	return s, nil
}

// InvalidateBlock marks a block invalid, forcing a reorg away from it (regtest reorg tests).
func (c *Client) InvalidateBlock(hash string) error {
	_, err := c.Call("invalidateblock", hash)
	return err
}

// ReconsiderBlock re-enables a previously invalidated block (restores the chain).
func (c *Client) ReconsiderBlock(hash string) error {
	_, err := c.Call("reconsiderblock", hash)
	return err
}

// GetBestBlockHash returns the current tip hash.
func (c *Client) GetBestBlockHash() (string, error) {
	raw, err := c.Call("getbestblockhash")
	if err != nil {
		return "", err
	}
	var s string
	return s, json.Unmarshal(raw, &s)
}

// SendRawTransaction broadcasts a raw tx hex and returns the txid.
func (c *Client) SendRawTransaction(txHex string) (string, error) {
	raw, err := c.Call("sendrawtransaction", txHex)
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", err
	}
	return s, nil
}
