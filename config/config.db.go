package config

import (
	"encoding/json"
	"fmt"
	"time"

	"gucooing/lolo/db"
)

type DB struct {
	Dev             bool      `toml:"dev"`
	DbType          db.DbType `json:"dbType"`
	Dsn             string    `json:"dsn"`     // 主库(写)
	ReadDsn         []string  `json:"readDsn"` // 从库(读),空=读写都走主库
	MaxIdleConns    int       `json:"maxIdleConns"`
	MaxOpenConns    int       `json:"maxOpenConns"`
	ConnMaxLifetime Duration  `json:"connMaxLifetime"`
	PersistShardNum int       `json:"persistShardNum"` // 异步落库分片数,0=CPU核数
	PersistBufSize  int       `json:"persistBufSize"`  // 每分片队列缓冲
}

var defaultDB = &DB{
	Dev:          false,
	DbType:       "sqlite",
	Dsn:          "./db/lolo.db?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)",
	ReadDsn:      make([]string, 0),
	MaxIdleConns: 20,
	MaxOpenConns: 40,
	ConnMaxLifetime: Duration{
		time.Hour,
	},
	PersistShardNum: 0,
	PersistBufSize:  1024,
}

func GetDB() *DB {
	if GetConfig().DB == nil {
		GetConfig().DB = defaultDB
	}
	return GetConfig().DB
}

func (x *DB) GetOption() *db.Option {
	return &db.Option{
		Dev:             x.Dev,
		Type:            x.DbType,
		Dsn:             x.Dsn,
		ReadDsn:         x.ReadDsn,
		MaxIdleConns:    x.MaxIdleConns,
		MaxOpenConns:    x.MaxOpenConns,
		ConnMaxLifetime: x.ConnMaxLifetime.Duration,
		PersistShardNum: x.PersistShardNum,
		PersistBufSize:  x.PersistBufSize,
	}
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value) * time.Second
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		return err
	default:
		return fmt.Errorf("无效的持续时间类型: %T", value)
	}
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}
