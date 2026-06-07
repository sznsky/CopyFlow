package uniswapv2

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// transferEventSig ERC20 Transfer(address,address,uint256) 事件签名。
var transferEventSig = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

// amountOutFromReceipt 从 receipt 的 Transfer 日志中解析买入代币数量。
func amountOutFromReceipt(receipt *types.Receipt, recipient, tokenOut common.Address) *big.Int {
	var total big.Int
	for _, lg := range receipt.Logs {
		if len(lg.Topics) != 3 || lg.Topics[0] != transferEventSig {
			continue
		}
		if lg.Address != tokenOut {
			continue
		}
		to := common.BytesToAddress(lg.Topics[2].Bytes())
		if to != recipient {
			continue
		}
		amount := new(big.Int).SetBytes(lg.Data)
		total.Add(&total, amount)
	}
	if total.Sign() == 0 {
		return big.NewInt(0)
	}
	return &total
}
