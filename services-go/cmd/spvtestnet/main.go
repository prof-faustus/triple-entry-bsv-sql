// spvtestnet validates the SPV/BURI proof layer (SYS-PROOF-*, Appendix B.10) against LIVE BSV TESTNET
// via WhatsOnChain (read-only; no funds). It fetches a real multi-tx block, reproduces its
// transaction-Merkle root with our spv package, and proves a real transaction's inclusion + BURI
// against the live block header's merkleroot.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"te-bsv/services-go/spv"
)

const api = "https://api.whatsonchain.com/v1/bsv/test"

var httpc = &http.Client{Timeout: 25 * time.Second}

func getJSON(path string, out any) error {
	for i := 0; i < 6; i++ {
		resp, err := httpc.Get(api + path)
		if err != nil {
			return err
		}
		if resp.StatusCode == 429 {
			resp.Body.Close()
			time.Sleep(1500 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("%s -> %d", path, resp.StatusCode)
		}
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return fmt.Errorf("%s rate-limited", path)
}

type chainInfo struct {
	Chain         string `json:"chain"`
	Blocks        int    `json:"blocks"`
	BestBlockHash string `json:"bestblockhash"`
}
type block struct {
	Hash       string   `json:"hash"`
	Height     int      `json:"height"`
	MerkleRoot string   `json:"merkleroot"`
	PrevHash   string   `json:"previousblockhash"`
	TxCount    int      `json:"txcount"`
	Tx         []string `json:"tx"`
}

func reverse(b []byte) []byte {
	out := make([]byte, len(b))
	for i := range b {
		out[len(b)-1-i] = b[i]
	}
	return out
}
func leafLE(txidBE string) []byte { b, _ := hex.DecodeString(txidBE); return reverse(b) }

func fail(msg string) { fmt.Println("RESULT: FAIL:", msg); os.Exit(1) }

func main() {
	var info chainInfo
	if err := getJSON("/chain/info", &info); err != nil {
		fail(err.Error())
	}
	fmt.Printf("LIVE BSV %s tip: height %d  %s\n", info.Chain, info.Blocks, info.BestBlockHash)

	// walk back from the tip to a block with a small, fully-listed multi-tx set
	hash := info.BestBlockHash
	var blk block
	for i := 0; i < 80; i++ {
		var b block
		if err := getJSON("/block/hash/"+hash, &b); err != nil {
			fail(err.Error())
		}
		if len(b.Tx) >= 2 && len(b.Tx) <= 200 {
			blk = b
			break
		}
		if b.PrevHash == "" {
			break
		}
		hash = b.PrevHash
		time.Sleep(250 * time.Millisecond)
	}
	if blk.Hash == "" {
		fail("no suitable multi-tx block found")
	}
	fmt.Printf("chosen block: height %d  %d txs  %s\n", blk.Height, len(blk.Tx), blk.Hash)

	// 1) reproduce the block's transaction-Merkle root with our spv package
	leaves := make([][]byte, len(blk.Tx))
	for i, txid := range blk.Tx {
		leaves[i] = leafLE(txid)
	}
	gotRootLE := spv.MerkleRoot(leaves)
	wantRootLE := leafLE(blk.MerkleRoot) // display -> LE
	if hex.EncodeToString(gotRootLE) != hex.EncodeToString(wantRootLE) {
		fail(fmt.Sprintf("merkle root mismatch:\n  got  %x\n  want(LE) %x", gotRootLE, wantRootLE))
	}
	fmt.Printf("merkle root reproduced from %d txids == live header merkleroot %s ✓\n", len(blk.Tx), blk.MerkleRoot)

	// 2) prove a real (non-coinbase) tx's inclusion + BURI against the live header root
	idx := len(blk.Tx) - 1 // last tx (non-coinbase when >1)
	branch, err := spv.BranchFor(leaves, idx)
	if err != nil {
		fail(err.Error())
	}
	if hex.EncodeToString(spv.RootFromBranch(leaves[idx], branch, idx)) != hex.EncodeToString(wantRootLE) {
		fail("branch does not reconstruct the live merkle root")
	}
	buri, err := spv.BuildBURI(blk.Hash, leaves, idx)
	if err != nil {
		fail(err.Error())
	}
	ok, err := buri.Verify(wantRootLE)
	if err != nil || !ok {
		fail("BURI did not SPV-verify against the live header root")
	}
	fmt.Printf("tx %s (index %d) inclusion proof + BURI SPV-verified against LIVE testnet header ✓\n", blk.Tx[idx], idx)
	fmt.Printf("BURI: %s\n", buri.String())
	fmt.Println("RESULT: SPV/BURI PASS on LIVE BSV TESTNET (SYS-PROOF-001..005, no funds needed)")
}
