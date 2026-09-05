// discover-wallets: 从 Uniswap V3 ETH/USDC 和 ETH/USDT 池子中
// 抓取过去 30 天内活跃的交易地址，按总交易量排序后取前 N 个，
// 写入 config.yaml 的 smartmoney.seed_wallets_uniswap 字段。
//
// 使用方法（在 backend/ 目录下执行）：
//   go run ./cmd/discover-wallets/
//   go run ./cmd/discover-wallets/ -config ./config/config.yaml -limit 10000
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ── 目标池子（Uniswap V3 Ethereum，小写地址）──────────────────────
var targetPools = []string{
	"0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640", // ETH/USDC 0.05%（主力池）
	"0x8ad599c3a0ff1de082011efddc58f1908eb6e6d5", // ETH/USDC 0.30%
	"0x4e68ccd3e89f51c3074ca5072bbac773960dfa36", // ETH/USDT 0.30%
	"0x11b815efb8f581194ae79006d24e0d814b7697f6", // ETH/USDT 0.05%
}

// 已知聚合器/路由合约黑名单（这些地址虽然出现在 origin，但并非真实用户）
var blacklist = map[string]bool{
	"0x1111111254eeb25477b68fb85ed929f73a960582": true, // 1inch v5
	"0x111111125421ca6dc452d289314280a0f8842a65": true, // 1inch v6
	"0xe592427a0aece92de3edee1f18e0157c05861564": true, // Uniswap Router V3
	"0x68b3465833fb72a70ecdf485e0e4c7bd8665fc45": true, // Uniswap Universal Router
	"0xdef1c0ded9bec7f1a1670819833240f027b25eff": true, // 0x Exchange Proxy
	"0xdef171fe48cf0115b1d80b88dc8eab59176fee57": true, // Paraswap
	"0x6131b5fae19ea4f9d964eac0408e4408b66337b5": true, // KyberSwap
}

// ── 数据结构 ─────────────────────────────────────────────────────
type swapData struct {
	ID        string `json:"id"`
	Origin    string `json:"origin"`
	AmountUSD string `json:"amountUSD"`
}

type graphResp struct {
	Data struct {
		Swaps []swapData `json:"swaps"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type traderStats struct {
	Address     string
	TotalVolume float64
	TradeCount  int
}

// httpClient 带超时，避免服务端挂起
var httpClient = &http.Client{Timeout: 60 * time.Second}

// ── The Graph 查询（含重试 + 指数退避）────────────────────────────
func fetchBatch(endpoint string, lastID string, startTS int64, minAmt float64, pools []string) ([]swapData, error) {
	poolJSON, _ := json.Marshal(pools)
	query := fmt.Sprintf(`{
  swaps(
    first: 1000
    orderBy: id
    orderDirection: asc
    where: {
      id_gt: "%s"
      timestamp_gte: "%d"
      amountUSD_gt: "%.0f"
      pool_in: %s
    }
  ) {
    id
    origin
    amountUSD
  }
}`, lastID, startTS, minAmt, string(poolJSON))

	body, _ := json.Marshal(map[string]string{"query": query})

	const maxRetries = 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		swaps, err := doRequest(endpoint, body)
		if err == nil {
			return swaps, nil
		}
		if attempt == maxRetries {
			return nil, fmt.Errorf("第 %d 次重试后仍失败: %w", maxRetries, err)
		}
		wait := time.Duration(attempt*attempt) * time.Second // 1s, 4s, 9s, 16s
		fmt.Printf("  ⚠️  请求失败（第 %d 次）: %v，%v 后重试...\n", attempt, err, wait)
		time.Sleep(wait)
	}
	return nil, fmt.Errorf("unreachable")
}

func doRequest(endpoint string, body []byte) ([]swapData, error) {
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (HTTP 429)")
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error (HTTP %d)", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result graphResp
	if err := json.Unmarshal(raw, &result); err != nil {
		preview := raw
		if len(preview) > 200 {
			preview = preview[:200]
		}
		return nil, fmt.Errorf("unmarshal: %w\nresponse: %s", err, string(preview))
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("graph error: %s", result.Errors[0].Message)
	}

	return result.Data.Swaps, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── YAML 更新 ─────────────────────────────────────────────────────
func updateConfig(configPath string, addresses []string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}
	if len(root.Content) == 0 {
		return fmt.Errorf("empty yaml document")
	}

	doc := root.Content[0] // 顶层 mapping

	// 找到 smartmoney 节点
	var smNode *yaml.Node
	for i := 0; i < len(doc.Content)-1; i += 2 {
		if doc.Content[i].Value == "smartmoney" {
			smNode = doc.Content[i+1]
			break
		}
	}
	if smNode == nil {
		return fmt.Errorf("smartmoney section not found in config")
	}

	// 构建地址序列节点
	seqNode := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.TaggedStyle}
	for _, addr := range addresses {
		seqNode.Content = append(seqNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: addr,
		})
	}

	// 查找已有的 seed_wallets_uniswap 并替换，否则追加
	found := false
	for i := 0; i < len(smNode.Content)-1; i += 2 {
		if smNode.Content[i].Value == "seed_wallets_uniswap" {
			smNode.Content[i+1] = seqNode
			found = true
			break
		}
	}
	if !found {
		keyNode := &yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "seed_wallets_uniswap",
			HeadComment: "# 由 discover-wallets 工具自动生成，请勿手动编辑此段",
		}
		smNode.Content = append(smNode.Content, keyNode, seqNode)
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ── 读取 The Graph endpoint from config ──────────────────────────
func readEndpointFromConfig(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	tg, ok := cfg["thegraph"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("thegraph section missing")
	}
	ep, ok := tg["uniswap_v3_endpoint"].(string)
	if !ok || ep == "" {
		return "", fmt.Errorf("uniswap_v3_endpoint missing")
	}
	return ep, nil
}

// ── 主流程 ────────────────────────────────────────────────────────
func main() {
	configPath := flag.String("config", "./config/config.yaml", "config.yaml 路径")
	limit := flag.Int("limit", 10000, "最多写入地址数量")
	days := flag.Int("days", 30, "统计最近 N 天")
	minAmt := flag.Float64("min-amount", 1000.0, "单笔最低交易金额（USD）")
	minTrades := flag.Int("min-trades", 5, "地址最低交易次数")
	flag.Parse()

	// ── 读取 endpoint ──
	endpoint, err := readEndpointFromConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 读取配置失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("📡 The Graph endpoint: %s\n", endpoint)

	startTS := time.Now().AddDate(0, 0, -*days).Unix()
	fmt.Printf("⏰ 统计周期: 最近 %d 天（%s 起）\n", *days, time.Unix(startTS, 0).Format("2006-01-02"))
	fmt.Printf("🔍 筛选条件: 单笔 ≥ $%.0f, 交易次数 ≥ %d\n", *minAmt, *minTrades)
	fmt.Printf("🎯 目标池子: %d 个（ETH/USDC + ETH/USDT，两个 fee tier）\n\n", len(targetPools))

	// ── 分页抓取所有 swap ──
	traders := make(map[string]*traderStats)
	lastID := ""
	totalFetched := 0
	batch := 0

	for {
		batch++
		swaps, err := fetchBatch(endpoint, lastID, startTS, *minAmt, targetPools)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 第 %d 批次请求失败: %v\n", batch, err)
			os.Exit(1)
		}
		if len(swaps) == 0 {
			break
		}

		for _, s := range swaps {
			origin := strings.ToLower(s.Origin)
			if origin == "" || blacklist[origin] {
				continue
			}
			amt, err := strconv.ParseFloat(s.AmountUSD, 64)
			if err != nil || amt < *minAmt {
				continue
			}
			if t, ok := traders[origin]; ok {
				t.TotalVolume += amt
				t.TradeCount++
			} else {
				traders[origin] = &traderStats{
					Address:     origin,
					TotalVolume: amt,
					TradeCount:  1,
				}
			}
			lastID = s.ID
		}

		totalFetched += len(swaps)
		if batch%10 == 0 || len(swaps) < 1000 {
			fmt.Printf("  批次 %4d | 已抓 %7d 条 | 唯一地址 %6d 个\n",
				batch, totalFetched, len(traders))
		}

		if len(swaps) < 1000 {
			break // 最后一批，结束
		}

		time.Sleep(300 * time.Millisecond) // 避免触发限速
	}

	fmt.Printf("\n✅ 抓取完成，共 %d 条 swap，%d 个唯一地址\n", totalFetched, len(traders))

	// ── 过滤：最低交易次数 ──
	var filtered []*traderStats
	for _, t := range traders {
		if t.TradeCount >= *minTrades {
			filtered = append(filtered, t)
		}
	}
	fmt.Printf("🔎 过滤后（交易次数 ≥ %d）：%d 个地址\n", *minTrades, len(filtered))

	// ── 排序：总交易量降序 ──
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].TotalVolume > filtered[j].TotalVolume
	})

	// ── 取前 limit 个 ──
	if len(filtered) > *limit {
		filtered = filtered[:*limit]
	}

	// ── 输出摘要 ──
	fmt.Printf("\n📊 Top 10 地址预览:\n")
	fmt.Printf("%-6s %-44s %15s %8s\n", "排名", "地址", "总交易量(USD)", "交易次数")
	fmt.Println(strings.Repeat("-", 80))
	for i, t := range filtered {
		if i >= 10 {
			break
		}
		fmt.Printf("%-6d %-44s %15.0f %8d\n", i+1, t.Address, t.TotalVolume, t.TradeCount)
	}

	// ── 写入 config.yaml ──
	addresses := make([]string, len(filtered))
	for i, t := range filtered {
		addresses[i] = t.Address
	}

	fmt.Printf("\n💾 正在写入 %s (seed_wallets_uniswap)...\n", *configPath)
	if err := updateConfig(*configPath, addresses); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 写入配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 成功写入 %d 个地址到 smartmoney.seed_wallets_uniswap\n", len(addresses))
	fmt.Println("ℹ️  原有 seed_wallets 不受影响")
}
