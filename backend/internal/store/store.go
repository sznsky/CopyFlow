// Package store 封装 MySQL 数据访问层。
package store

import (
	"strings"

	"copyflow/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Store 数据库操作入口。
type Store struct {
	db *gorm.DB
}

// New 创建 Store 并连接 MySQL。
func New(dsn string) (*Store, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// DB 返回底层 gorm 实例，供特殊查询使用。
func (s *Store) DB() *gorm.DB {
	return s.db
}

// AutoMigrate 自动同步表结构到数据库。
func (s *Store) AutoMigrate() error {
	return s.db.AutoMigrate(
		&model.User{},
		&model.EmailVerification{},
		&model.CopyWallet{},
		&model.CopyConfig{},
		&model.LeaderTrade{},
		&model.CopyTrade{},
		&model.ChainCursor{},
		// Smart Money models
		&model.SmartWallet{},
		&model.WalletTrade{},
		&model.TokenSignal{},
		&model.TokenSignalDetail{},
		&model.SyncLog{},
	)
}

// --- User ---

// GetUserByAddress 按钱包地址查询用户。
func (s *Store) GetUserByAddress(address string) (*model.User, error) {
	var u model.User
	err := s.db.Where("wallet_address = ?", strings.ToLower(address)).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID 按用户 ID 查询。
func (s *Store) GetUserByID(id uint64) (*model.User, error) {
	var u model.User
	err := s.db.First(&u, id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpsertUserNonce 创建或更新用户登录 nonce。
func (s *Store) UpsertUserNonce(address, nonce string) (*model.User, error) {
	addr := strings.ToLower(address)
	var u model.User
	err := s.db.Where("wallet_address = ?", addr).First(&u).Error
	if err == gorm.ErrRecordNotFound {
		wa := addr
		u = model.User{WalletAddress: &wa, Nonce: nonce}
		if err := s.db.Create(&u).Error; err != nil {
			return nil, err
		}
		return &u, nil
	}
	if err != nil {
		return nil, err
	}
	u.Nonce = nonce
	if err := s.db.Save(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// --- CopyWallet ---

// CreateCopyWallet 保存新生成的跟单钱包。
func (s *Store) CreateCopyWallet(w *model.CopyWallet) error {
	return s.db.Create(w).Error
}

// HasCopyWallet 检查用户在某链是否已有跟单钱包。
func (s *Store) HasCopyWallet(userID uint64, chainID int) (bool, error) {
	var count int64
	err := s.db.Model(&model.CopyWallet{}).
		Where("user_id = ? AND chain_id = ?", userID, chainID).
		Count(&count).Error
	return count > 0, err
}

// GetCopyWallet 获取用户在指定链上的活跃跟单钱包。
func (s *Store) GetCopyWallet(userID uint64, chainID int) (*model.CopyWallet, error) {
	var w model.CopyWallet
	err := s.db.Where("user_id = ? AND chain_id = ? AND is_active = ?", userID, chainID, true).First(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// ListCopyWallets 列出用户所有跟单钱包。
func (s *Store) ListCopyWallets(userID uint64) ([]model.CopyWallet, error) {
	var list []model.CopyWallet
	err := s.db.Where("user_id = ?", userID).Find(&list).Error
	return list, err
}

// --- CopyConfig ---

// CreateCopyConfig 新增跟单配置。
func (s *Store) CreateCopyConfig(c *model.CopyConfig) error {
	c.LeaderAddress = strings.ToLower(c.LeaderAddress)
	return s.db.Create(c).Error
}

// ListCopyConfigs 列出用户的跟单配置。
func (s *Store) ListCopyConfigs(userID uint64) ([]model.CopyConfig, error) {
	var list []model.CopyConfig
	err := s.db.Where("user_id = ?", userID).Order("id DESC").Find(&list).Error
	return list, err
}

// GetCopyConfigByID 按 ID 查询配置（Worker 内部使用）。
func (s *Store) GetCopyConfigByID(id uint64) (*model.CopyConfig, error) {
	var c model.CopyConfig
	err := s.db.First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetCopyConfig 按用户 ID 和配置 ID 查询（带权限校验）。
func (s *Store) GetCopyConfig(userID, id uint64) (*model.CopyConfig, error) {
	var c model.CopyConfig
	err := s.db.Where("user_id = ? AND id = ?", userID, id).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// UpdateCopyConfig 更新跟单配置。
func (s *Store) UpdateCopyConfig(userID uint64, c *model.CopyConfig) error {
	c.LeaderAddress = strings.ToLower(c.LeaderAddress)
	return s.db.Where("user_id = ? AND id = ?", userID, c.ID).Updates(c).Error
}

// DeleteCopyConfig 删除跟单配置。
func (s *Store) DeleteCopyConfig(userID, id uint64) error {
	return s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&model.CopyConfig{}).Error
}

// ListActiveCopyConfigs 列出所有启用的跟单配置。
func (s *Store) ListActiveCopyConfigs() ([]model.CopyConfig, error) {
	var list []model.CopyConfig
	err := s.db.Where("is_active = ?", true).Find(&list).Error
	return list, err
}

// ListActiveConfigsByLeader 查询监听某领头地址的所有活跃配置。
func (s *Store) ListActiveConfigsByLeader(chainID int, leader string) ([]model.CopyConfig, error) {
	var list []model.CopyConfig
	err := s.db.Where("is_active = ? AND chain_id = ? AND leader_address = ?",
		true, chainID, strings.ToLower(leader)).Find(&list).Error
	return list, err
}

// --- LeaderTrade ---

// CreateLeaderTradeIfNotExists 幂等写入领头交易，返回是否为新记录。
func (s *Store) CreateLeaderTradeIfNotExists(t *model.LeaderTrade) (*model.LeaderTrade, bool, error) {
	t.LeaderAddress = strings.ToLower(t.LeaderAddress)
	var existing model.LeaderTrade
	err := s.db.Where("chain_id = ? AND tx_hash = ?", t.ChainID, t.TxHash).First(&existing).Error
	if err == nil {
		return &existing, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}
	if err := s.db.Create(t).Error; err != nil {
		return nil, false, err
	}
	return t, true, nil
}

// ListLeaderTrades 分页查询领头交易记录。
func (s *Store) ListLeaderTrades(limit int) ([]model.LeaderTrade, error) {
	var list []model.LeaderTrade
	q := s.db.Order("id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&list).Error
	return list, err
}

// GetLeaderTrade 按 ID 查询领头交易。
func (s *Store) GetLeaderTrade(id uint64) (*model.LeaderTrade, error) {
	var t model.LeaderTrade
	err := s.db.First(&t, id).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// --- CopyTrade ---

// CreateCopyTrade 写入跟单执行记录。
func (s *Store) CreateCopyTrade(t *model.CopyTrade) error {
	return s.db.Create(t).Error
}

// UpdateCopyTrade 更新跟单记录状态或 tx hash。
func (s *Store) UpdateCopyTrade(t *model.CopyTrade) error {
	return s.db.Save(t).Error
}

// ListSubmittedCopyTrades 查询已广播待确认的跟单交易。
func (s *Store) ListSubmittedCopyTrades(limit int) ([]model.CopyTrade, error) {
	var list []model.CopyTrade
	q := s.db.Where("status = ? AND tx_hash IS NOT NULL", model.TradeStatusSubmitted).Order("id ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&list).Error
	return list, err
}

// ListCopyTrades 查询用户的跟单记录（含关联领头交易）。
func (s *Store) ListCopyTrades(userID uint64, limit int) ([]model.CopyTrade, error) {
	var list []model.CopyTrade
	q := s.db.Where("user_id = ?", userID).Preload("LeaderTrade").Order("id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&list).Error
	return list, err
}

// --- ChainCursor ---

// GetChainCursor 获取链上扫描游标（已处理区块高度）。
func (s *Store) GetChainCursor(chainID int) (uint64, error) {
	var c model.ChainCursor
	err := s.db.Where("chain_id = ?", chainID).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return c.LastBlock, nil
}

// SetChainCursor 更新链上扫描游标。
func (s *Store) SetChainCursor(chainID int, block uint64) error {
	var c model.ChainCursor
	err := s.db.Where("chain_id = ?", chainID).First(&c).Error
	if err == gorm.ErrRecordNotFound {
		c = model.ChainCursor{ChainID: chainID, LastBlock: block}
		return s.db.Create(&c).Error
	}
	if err != nil {
		return err
	}
	c.LastBlock = block
	return s.db.Save(&c).Error
}

// DistinctActiveLeaders 返回各链上需要监听的领头地址（去重）。
func (s *Store) DistinctActiveLeaders() (map[int][]string, error) {
	type row struct {
		ChainID       int
		LeaderAddress string
	}
	var rows []row
	err := s.db.Model(&model.CopyConfig{}).
		Select("chain_id, leader_address").
		Where("is_active = ?", true).
		Group("chain_id, leader_address").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int][]string)
	for _, r := range rows {
		result[r.ChainID] = append(result[r.ChainID], strings.ToLower(r.LeaderAddress))
	}
	return result, nil
}

// --- Email Auth ---

// GetUserByEmail 按邮箱查询用户。
func (s *Store) GetUserByEmail(email string) (*model.User, error) {
	var u model.User
	err := s.db.Where("email = ?", strings.ToLower(email)).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateEmailUser 创建邮箱注册用户。
func (s *Store) CreateEmailUser(email, passwordHash string) (*model.User, error) {
	em := strings.ToLower(email)
	u := &model.User{Email: &em, PasswordHash: passwordHash}
	if err := s.db.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// CreateEmailVerification 保存邮箱验证码。
func (s *Store) CreateEmailVerification(v *model.EmailVerification) error {
	v.Email = strings.ToLower(v.Email)
	return s.db.Create(v).Error
}

// FindValidEmailCode 查找未过期的验证码。
func (s *Store) FindValidEmailCode(email, code, purpose string) (*model.EmailVerification, error) {
	var v model.EmailVerification
	err := s.db.Where(
		"email = ? AND code = ? AND purpose = ? AND used = ? AND expires_at > NOW(3)",
		strings.ToLower(email), code, purpose, false,
	).Order("id DESC").First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// MarkEmailCodeUsed 标记验证码已使用。
func (s *Store) MarkEmailCodeUsed(id uint64) error {
	return s.db.Model(&model.EmailVerification{}).Where("id = ?", id).Update("used", true).Error
}
