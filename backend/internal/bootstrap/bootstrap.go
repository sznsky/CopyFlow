// Package bootstrap 根据配置初始化链客户端与 DEX 注册表。
package bootstrap

import (
	"fmt"
	"log"

	"copyflow/internal/chain"
	"copyflow/internal/chain/evm"
	"copyflow/internal/config"
	"copyflow/internal/dex"
	"copyflow/internal/dex/uniswapv2"
	"copyflow/internal/model"
)

// BuildChains 从配置创建链注册表。
func BuildChains(cfg *config.Config) (*chain.Registry, error) {
	reg := chain.NewRegistry()
	for _, ch := range cfg.Chains {
		if !ch.Enabled {
			log.Printf("[bootstrap] chain %s disabled, skipping", ch.Name)
			continue
		}
		client, err := evm.NewClient(ch)
		if err != nil {
			return nil, fmt.Errorf("init chain %s: %w", ch.Name, err)
		}
		reg.Register(client)
		log.Printf("[bootstrap] chain %s (id=%d) registered", ch.Name, ch.ChainID)
	}
	return reg, nil
}

// BuildDEXRegistry 从配置注册 DEX Parser 与 Executor。
func BuildDEXRegistry(cfg *config.Config) (*dex.Registry, error) {
	reg := dex.NewRegistry()
	for _, ch := range cfg.Chains {
		if !ch.Enabled {
			continue
		}
		for _, d := range ch.DEXes {
			if !d.Enabled {
				continue
			}
			switch d.Type {
			case model.DEXPancakeV2, model.DEXUniswapV2:
				parser, err := uniswapv2.NewParser(d.Type, d.RouterAddress, d.WrappedNative)
				if err != nil {
					return nil, err
				}
				exec, err := uniswapv2.NewExecutor(d.Type, d.RouterAddress, d.WrappedNative)
				if err != nil {
					return nil, err
				}
				reg.RegisterParser(ch.ChainID, parser)
				reg.RegisterExecutor(exec)
				log.Printf("[bootstrap] dex %s on chain %d registered", d.Type, ch.ChainID)
			case model.DEXUniswapV3:
				log.Printf("[bootstrap] dex %s not implemented in MVP, skipped", d.Type)
			default:
				log.Printf("[bootstrap] unknown dex type %s", d.Type)
			}
		}
	}
	return reg, nil
}
