package db

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"

	"gorm.io/gorm"
	gromlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
)

var (
	db       *gorm.DB
	noDbType = errors.New("不正确的DbType")
)

type DbType string

const (
	Sqlite   DbType = "sqlite"
	Mysql    DbType = "mysql"
	Postgres DbType = "postgres"
)

type Option struct {
	Dev             bool          // 是否调试
	Type            DbType        // 数据库类型
	Dsn             string        // 数据库地址(主库,写)
	ReadDsn         []string      // 从库(读)地址,空=读写都走主库
	MaxIdleConns    int           // 最大空闲连接数
	MaxOpenConns    int           // 最大连接数
	ConnMaxLifetime time.Duration // 最大连接复用时间
	PersistShardNum int           // 异步落库分片数,0=CPU核数
	PersistBufSize  int           // 每分片队列缓冲
}

type Database struct {
	option *Option
	db     *gorm.DB
}

func NewDB(option *Option) error {
	d := &Database{option: option}
	var err error
	switch option.Type {
	case Mysql:
		err = d.newMysql()
	case Sqlite:
		err = d.newSqlite()
	case Postgres:
		err = d.newPostgres()
	default:
		return noDbType
	}
	if err != nil {
		return err
	}
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}

	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(d.option.MaxIdleConns)
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(d.option.MaxOpenConns)
	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	sqlDB.SetConnMaxLifetime(d.option.ConnMaxLifetime)

	// 读写分离:配置了从库才注册 dbresolver(读→从库,写→主库)
	if len(d.option.ReadDsn) > 0 {
		if err = d.useReadWriteSplit(); err != nil {
			return err
		}
	}

	time1 := time.Now()
	err = d.db.AutoMigrate(
		&OFQuick{},
		&OFUser{},
		&OFQuickCheck{},
		&OFGame{},
		&OFGameBasic{},
		&BlackDevice{},
		&OFFriendInfo{},
		&OFFriend{},
		&OFFriendBlack{},
		&OFChatPrivate{},
		&OFChatPrivateMsg{},
		&OFGachaRecord{},
		&OFHome{},
	)

	if time.Now().Sub(time1) >= 2*time.Second {
		log.Printf("数据库迁移耗时过长: %s 建议更换本地数据库", time.Now().Sub(time1).String())
	}

	db = d.db

	// 启动异步落库分片写池
	StartPersist(resolveShardNum(d.option.PersistShardNum, d.option.MaxOpenConns), d.option.PersistBufSize)

	return err
}

// Close 关闭数据库:先排空异步落库队列,再关闭连接。关服时调用。
func Close() error {
	StopPersist()
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// dialector 按当前数据库类型为给定 dsn 构造 gorm 方言
func (d *Database) dialector(dsn string) gorm.Dialector {
	switch d.option.Type {
	case Mysql:
		return mysql.Open(dsn)
	case Postgres:
		return postgres.Open(dsn)
	case Sqlite:
		return sqlite.Open(dsn)
	}
	return nil
}

// useReadWriteSplit 注册 dbresolver:读走从库,写走主库
func (d *Database) useReadWriteSplit() error {
	replicas := make([]gorm.Dialector, 0, len(d.option.ReadDsn))
	for _, dsn := range d.option.ReadDsn {
		replicas = append(replicas, d.dialector(dsn))
	}
	return d.db.Use(
		dbresolver.Register(dbresolver.Config{
			Sources:  []gorm.Dialector{d.dialector(d.option.Dsn)},
			Replicas: replicas,
			Policy:   dbresolver.RandomPolicy{},
		}).
			SetMaxIdleConns(d.option.MaxIdleConns).
			SetMaxOpenConns(d.option.MaxOpenConns).
			SetConnMaxLifetime(d.option.ConnMaxLifetime),
	)
}

// resolveShardNum 落库分片数:0→CPU核数,且不超过最大连接数
func resolveShardNum(n, maxConns int) int {
	if n <= 0 {
		n = runtime.NumCPU()
	}
	if maxConns > 0 && n > maxConns {
		n = maxConns
	}
	if n < 1 {
		n = 1
	}
	return n
}

func (d *Database) newMysql() error {
	openDb, err := gorm.Open(mysql.Open(d.option.Dsn), d.getGormConfig())
	if err != nil {
		return err
	}
	d.db = openDb
	return nil
}

func (d *Database) newSqlite() error {
	if _, err := os.Stat(filepath.Dir(d.option.Dsn)); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(d.option.Dsn), 0777)
		if err != nil {
			return err
		}
	}
	openDb, err := gorm.Open(sqlite.Open(d.option.Dsn), d.getGormConfig())
	if err != nil {
		return err
	}
	d.db = openDb
	return nil
}

func (d *Database) newPostgres() error {
	openDb, err := gorm.Open(postgres.Open(d.option.Dsn), d.getGormConfig())
	if err != nil {
		return err
	}
	d.db = openDb
	return nil
}

func (d *Database) getGormConfig() *gorm.Config {
	info := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	}
	if d.option.Dev {
		info.Logger = gromlogger.Default.LogMode(gromlogger.Info)
	} else {
		info.Logger = gromlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gromlogger.Config{
				SlowThreshold: time.Second,
				LogLevel:      gromlogger.Warn,
				Colorful:      false,
			},
		)
	}
	return info
}
