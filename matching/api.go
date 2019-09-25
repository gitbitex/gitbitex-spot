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

package matching

import (
	"github.com/gitbitex/gitbitex-spot/models"
)

// 用于撮合引擎读取order，需要支持设置offset，从指定的offset开始读取
type OrderReader interface {
	// 设置读取的起始offset
	SetOffset(offset int64) error

	// 拉取order
	FetchOrder() (offset int64, order *models.Order, err error)
}

// 用于保存撮合日志
type LogStore interface {
	// 保存日志
	Store(logs []interface{}) error
}

// 以观察者模式读取撮合日志
type LogReader interface {
	// 获取当前的productId
	GetProductId() string

	// 注册一个日志观察者
	RegisterObserver(observer LogObserver)

	// 开始执行读取log，读取到的log将会回调给观察者
	Run(seq, offset int64)
}

// 撮合日志reader观察者
type LogObserver interface {
	// 当读到OpenLog时回调
	OnOpenLog(log *OpenLog, offset int64)

	// 当读到MatchLog时回调
	OnMatchLog(log *MatchLog, offset int64)

	// 当读到DoneLog是回调
	OnDoneLog(log *DoneLog, offset int64)
}

// 用于保存撮合引擎的快照
type SnapshotStore interface {
	// 保存快照
	Store(snapshot *Snapshot) error

	// 获取最后一次快照
	GetLatest() (*Snapshot, error)
}
