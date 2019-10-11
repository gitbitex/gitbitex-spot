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

package service

import (
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/models/mysql"
)

func GetLastTickByProductId(productId string, granularity int64) (*models.Tick, error) {
	return mysql.SharedStore().GetLastTickByProductId(productId, granularity)
}

func GetTicksByProductId(productId string, granularity int64, limit int) ([]*models.Tick, error) {
	return mysql.SharedStore().GetTicksByProductId(productId, granularity, limit)
}

func AddTicks(ticks []*models.Tick) error {
	if len(ticks) == 0 {
		return nil
	}
	return mysql.SharedStore().AddTicks(ticks)
}
