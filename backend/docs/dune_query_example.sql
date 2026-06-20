-- Dune Analytics 查询示例：Uniswap V2 交易数据
-- 用于聪明钱分析系统
-- 
-- 查询目标：
-- 1. 获取 USDC/USDT、ETH/USDT 池子的 Swap 交易
-- 2. 金额 > 1000 USD
-- 3. 包含完整的交易信息（买入/卖出、代币、数量等）
-- 
-- 注意：此查询需要根据实际 Dune 数据表结构调整

-- 参数（在 Dune 界面中定义）
-- @min_amount_usd: 最小交易金额（默认 1000）
-- @days_back: 回溯天数（默认 180，即6个月）

WITH 
-- 定义目标代币地址（Ethereum 主网）
target_tokens AS (
    SELECT 0xdac17f958d2ee523a2206206994597c13d831ec7 AS address, 'USDT' AS symbol, 6 AS decimals -- USDT
    UNION ALL
    SELECT 0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48, 'USDC', 6 -- USDC
    UNION ALL
    SELECT 0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2, 'WETH', 18 -- WETH
    UNION ALL
    SELECT 0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee, 'ETH', 18 -- ETH (native)
),

-- Uniswap V2 交易数据
uniswap_v2_swaps AS (
    SELECT
        evt_block_time AS block_time,
        evt_block_number AS block_number,
        evt_tx_hash AS tx_hash,
        "to" AS wallet_address,
        contract_address AS pool_address,
        
        -- 处理 Swap 事件的输入/输出
        CASE 
            WHEN amount0In > 0 THEN pair.token0
            ELSE pair.token1
        END AS token_in,
        
        CASE 
            WHEN amount0Out > 0 THEN pair.token0
            ELSE pair.token1
        END AS token_out,
        
        CASE 
            WHEN amount0In > 0 THEN amount0In
            ELSE amount1In
        END AS amount_in_raw,
        
        CASE 
            WHEN amount0Out > 0 THEN amount0Out
            ELSE amount1Out
        END AS amount_out_raw,
        
        'uniswap_v2' AS dex_name
        
    FROM uniswap_v2_ethereum.Pair_evt_Swap AS swaps
    INNER JOIN uniswap_v2_ethereum.Factory_evt_PairCreated AS pair
        ON swaps.contract_address = pair.pair
    WHERE evt_block_time >= NOW() - INTERVAL '{{days_back}}' DAY
        AND evt_block_time <= NOW()
),

-- 关联代币信息和价格
enriched_swaps AS (
    SELECT
        s.block_time,
        s.block_number,
        s.tx_hash,
        s.wallet_address,
        s.pool_address,
        s.dex_name,
        
        s.token_in,
        s.token_out,
        
        t_in.symbol AS token_in_symbol,
        t_out.symbol AS token_out_symbol,
        
        s.amount_in_raw / POWER(10, COALESCE(t_in.decimals, 18)) AS amount_in,
        s.amount_out_raw / POWER(10, COALESCE(t_out.decimals, 18)) AS amount_out,
        
        -- 计算 USD 价值（使用 Dune 的价格表）
        (s.amount_in_raw / POWER(10, COALESCE(t_in.decimals, 18))) * p_in.price AS amount_in_usd,
        (s.amount_out_raw / POWER(10, COALESCE(t_out.decimals, 18))) * p_out.price AS amount_out_usd
        
    FROM uniswap_v2_swaps AS s
    
    LEFT JOIN tokens.erc20 AS t_in
        ON s.token_in = t_in.contract_address
        AND t_in.blockchain = 'ethereum'
    
    LEFT JOIN tokens.erc20 AS t_out
        ON s.token_out = t_out.contract_address
        AND t_out.blockchain = 'ethereum'
    
    LEFT JOIN prices.usd AS p_in
        ON s.token_in = p_in.contract_address
        AND p_in.blockchain = 'ethereum'
        AND DATE_TRUNC('minute', s.block_time) = p_in.minute
    
    LEFT JOIN prices.usd AS p_out
        ON s.token_out = p_out.contract_address
        AND p_out.blockchain = 'ethereum'
        AND DATE_TRUNC('minute', s.block_time) = p_out.minute
),

-- 判断买入/卖出方向
classified_swaps AS (
    SELECT
        *,
        
        -- 判断是否为买入（买入目标代币，卖出稳定币/ETH）
        CASE
            WHEN token_out IN (SELECT address FROM target_tokens WHERE symbol IN ('USDT', 'USDC', 'WETH', 'ETH'))
                 AND token_in NOT IN (SELECT address FROM target_tokens WHERE symbol IN ('USDT', 'USDC', 'WETH', 'ETH'))
            THEN false  -- 卖出
            WHEN token_in IN (SELECT address FROM target_tokens WHERE symbol IN ('USDT', 'USDC', 'WETH', 'ETH'))
                 AND token_out NOT IN (SELECT address FROM target_tokens WHERE symbol IN ('USDT', 'USDC', 'WETH', 'ETH'))
            THEN true  -- 买入
            ELSE NULL  -- 稳定币之间或无法判断
        END AS is_buy,
        
        -- 使用输入金额作为交易金额（更准确）
        GREATEST(amount_in_usd, amount_out_usd) AS amount_usd
        
    FROM enriched_swaps
    WHERE amount_in_usd > 0 OR amount_out_usd > 0  -- 确保有价格数据
)

-- 最终筛选和输出
SELECT
    wallet_address,
    tx_hash,
    block_number,
    block_time,
    dex_name,
    CASE
        WHEN dex_name LIKE '%v3%' THEN 'v3'
        WHEN dex_name LIKE '%v2%' THEN 'v2'
        ELSE NULL
    END AS dex_version,
    pool_address,
    token_in,
    token_out,
    token_in_symbol,
    token_out_symbol,
    CAST(amount_in AS VARCHAR) AS amount_in,
    CAST(amount_out AS VARCHAR) AS amount_out,
    amount_usd,
    is_buy,
    
    -- 预留字段（需要复杂的逻辑计算，建议在后端处理）
    NULL AS pnl_usd,
    NULL AS pnl_percent,
    NULL AS holding_duration_hours

FROM classified_swaps

WHERE 
    -- 只保留能明确判断方向的交易
    is_buy IS NOT NULL
    
    -- 金额过滤
    AND amount_usd >= {{min_amount_usd}}
    
    -- 排除合约地址（可选）
    AND wallet_address != 0x0000000000000000000000000000000000000000
    
    -- 只保留与目标池子相关的交易
    AND (
        token_in IN (SELECT address FROM target_tokens)
        OR token_out IN (SELECT address FROM target_tokens)
    )

ORDER BY block_time DESC

LIMIT 10000  -- 限制结果数量，避免超时

-- 注意事项：
-- 1. 此查询是示例，需要根据 Dune 的实际数据表名和字段名调整
-- 2. Dune 的表名格式通常为：{protocol}_{blockchain}.{table_name}
-- 3. 价格数据可能有延迟或缺失，建议在后端再次验证
-- 4. 对于不同链，需要创建单独的查询（修改 blockchain 参数）
-- 5. 可以添加更多过滤条件，如 MEV bot 检测、Gas 价格过滤等
