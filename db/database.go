package db

import (
	"log"
	"os"
	"time"

	"github.com/juliotorresmoreno/doppler/config"
	"github.com/juliotorresmoreno/doppler/model"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func GetConnection() (*gorm.DB, error) {
	var err error
	if db != nil {
		return db, nil
	}
	db, err = makeConnection()
	return db, err
}

func makeConnection() (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	conf, _ := config.GetConfig()
	dbConf := conf.Database
	switch dbConf["driver"] {
	case "postgres":
		db, err = makePostgreSQLConnection(dbConf)
	case "sqlite":
		fallthrough
	default:
		db, err = makeSQLiteConnection(dbConf)
	}

	if err != nil {
		return db, err
	}

	migrate(db)
	return db, nil
}

func makeSQLiteConnection(dbConf map[string]string) (*gorm.DB, error) {
	path := dbConf["path"]
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	return db, err
}

func makePostgreSQLConnection(dbConf map[string]string) (*gorm.DB, error) {
	dsn := dbConf["dsn"]
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Silent,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			Colorful:                  false,
		},
	)
	db.Logger = newLogger

	return db, err
}

func migrate(db *gorm.DB) {
	db.AutoMigrate(&model.Log{})
	db.AutoMigrate(&model.User{})
	db.AutoMigrate(&model.Server{})
}
