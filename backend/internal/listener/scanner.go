// Package listener 扫描链上区块，发现领头地址的 DEX 买入交易。
package listener

import (
	"context"
	"log"
	"math/big"
	"strings"

	"copyflow/internal/chain"
	"copyflow/internal/dex"
	"copyflow/internal/store"

	"github.com/ethereum/go-ethereum/core/types"
)

// Scanner 按区块扫描领头地址交易。
// 扩展点：WebSocket 订阅、mempool 监听、第三方 indexer 回调。
type Scanner struct {
	chains  *chain.Registry
	dexes   *dex.Registry
	store   *store.Store
	leaders map[int]map[string]struct{}
}

// NewScanner 创建区块扫描器。
func NewScanner(chains *chain.Registry, dexes *dex.Registry, st *store.Store) *Scanner {
	return &Scanner{
		chains:  chains,
		dexes:   dexes,
		store:   st,
		leaders: make(map[int]map[string]struct{}),
	}
}

// RefreshLeaders 从数据库刷新需要监听的领头地址列表。
func (s *Scanner) RefreshLeaders(ctx context.Context) error {
	leaders, err := s.store.DistinctActiveLeaders()
	if err != nil {
		return err
	}
	m := make(map[int]map[string]struct{})
	for chainID, addrs := range leaders {
		m[chainID] = make(map[string]struct{})
		for _, a := range addrs {
			m[chainID][strings.ToLower(a)] = struct{}{}
		}
	}
	s.leaders = m
	return nil
}

// isLeader 判断地址是否为当前需要监听的领头地址。
func (s *Scanner) isLeader(chainID int, addr string) bool {
	set, ok := s.leaders[chainID]
	if !ok {
		return false
	}
	_, ok = set[strings.ToLower(addr)]
	return ok
}

// LeaderTxMeta 扫描到的领头地址买入交易元数据。
type LeaderTxMeta struct {
	ChainID       int
	LeaderAddress string
	TxHash        string
	BlockNumber   uint64
	Swap          *dex.SwapInfo
}

// ScanChain 从上次游标扫描到 latest-confirmations，返回发现的买入交易。
func (s *Scanner) ScanChain(ctx context.Context, chainID int, confirmations uint64) ([]LeaderTxMeta, error) {
	client, ok := s.chains.Get(chainID)
	if !ok {
		return nil, nil
	}

	latest, err := client.BlockNumber(ctx)
	if err != nil {
		return nil, err
	}
	if latest <= confirmations {
		return nil, nil
	}
	target := latest - confirmations

	lastBlock, err := s.store.GetChainCursor(chainID)
	if err != nil {
		return nil, err
	}
	if lastBlock == 0 && target > 10 {
		lastBlock = target - 10
	}

	var metas []LeaderTxMeta

	for bn := lastBlock + 1; bn <= target; bn++ {
		block, err := client.BlockByNumber(ctx, bn)
		if err != nil {
			log.Printf("[listener] block %d error: %v", bn, err)
			continue
		}
		chainIDBig := big.NewInt(int64(chainID))
		signer := types.LatestSignerForChainID(chainIDBig)

		for _, tx := range block.Transactions() {
			from, err := types.Sender(signer, tx)
			if err != nil {
				continue
			}
			if !s.isLeader(chainID, from.Hex()) {
				continue
			}
			if tx.To() == nil {
				continue
			}

			parser, ok := s.dexes.ParserByRouter(chainID, *tx.To())
			if !ok {
				continue
			}

			receipt, err := client.TransactionReceipt(ctx, tx.Hash())
			if err != nil || receipt.Status != 1 {
				continue
			}

			swap, err := parser.ParseSwap(ctx, client, tx, receipt)
			if err != nil || swap == nil || !swap.IsBuy {
				continue
			}

			metas = append(metas, LeaderTxMeta{
				ChainID:       chainID,
				LeaderAddress: strings.ToLower(from.Hex()),
				TxHash:        tx.Hash().Hex(),
				BlockNumber:   bn,
				Swap:          swap,
			})
		}
	}

	if target > lastBlock {
		if err := s.store.SetChainCursor(chainID, target); err != nil {
			log.Printf("[listener] cursor update error: %v", err)
		}
	}

	return metas, nil
}
