// Copyright 2019 GitBitEx.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mysql

import (
	"fmt"
	"github.com/gitbitex/gitbitex-spot/conf"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/jinzhu/gorm"
	"sync"
)

var gdb *gorm.DB
var store models.Store
var storeOnce sync.Once

type Store struct {
	db *gorm.DB
}

func SharedStore() models.Store {
	storeOnce.Do(func() {
		err := initDb()
		if err != nil {
			panic(err)
		}
		store = NewStore(gdb)
	})
	return store
}

func NewStore(db *gorm.DB) *Store {
	return &Store{
		db: db,
	}
}

func initDb() error {
	cfg, err := conf.GetConfig()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%v:%v@tcp(%v)/%v?charset=utf8&parseTime=True&loc=Local",
		cfg.DataSource.User, cfg.DataSource.Password, cfg.DataSource.Addr, cfg.DataSource.Database)
	gdb, err = gorm.Open(cfg.DataSource.DriverName, url)
	if err != nil {
		return err
	}

	gdb.SingularTable(true)
	gdb.DB().SetMaxIdleConns(10)
	gdb.DB().SetMaxOpenConns(50)

	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return "g_" + defaultTableName
	}

	/*	log.Error(gdb.AutoMigrate(&models.Account{}).Error)
		log.Error(gdb.AutoMigrate(&models.Order{}).Error)
		log.Error(gdb.AutoMigrate(&models.Product{}).Error)
		log.Error(gdb.AutoMigrate(&models.Trade{}).Error)
		log.Error(gdb.AutoMigrate(&models.Fill{}).Error)
		log.Error(gdb.AutoMigrate(&models.User{}).Error)
		log.Error(gdb.AutoMigrate(&models.Bill{}).Error)
		log.Error(gdb.AutoMigrate(&models.Tick{}).Error)
		log.Error(gdb.AutoMigrate(&models.Config{}).Error)
	*/
	return nil
}

func (s *Store) BeginTx() (models.Store, error) {
	db := s.db.Begin()
	if db.Error != nil {
		return nil, db.Error
	}
	return NewStore(db), nil
}

func (s *Store) Rollback() error {
	return s.db.Rollback().Error
}

func (s *Store) CommitTx() error {
	return s.db.Commit().Error
}
