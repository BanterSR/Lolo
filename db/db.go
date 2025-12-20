package db

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"

	"gorm.io/gorm"
	gromlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
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
	Dsn             string        // 数据库地址
	MaxIdleConns    int           // 最大空闲连接数
	MaxOpenConns    int           // 最大连接数
	ConnMaxLifetime time.Duration // 最大连接复用时间
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

	err = d.db.AutoMigrate(
		&OFUser{},
		&OFGame{},
		&OFGameBasic{},
		&BlackDevice{},
		&OFFriendInfo{},
		&OFFriendRequest{},
		&OFFriend{},
		&OFFriendBlack{},
		&OFChatPrivate{},
		&OFChatPrivateMsg{},
		&OFGachaRecord{},
	)

	d.db.FirstOrCreate(&OFUser{
		UserId: 999999,
		SdkUid: 0,
	})

	db = d.db

	return err
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
		os.MkdirAll(filepath.Dir(d.option.Dsn), 0777)
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
