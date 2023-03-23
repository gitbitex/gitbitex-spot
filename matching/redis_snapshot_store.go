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
	"encoding/json"
	"github.com/gitbitex/gitbitex-spot/conf"
	"github.com/go-redis/redis"
	"time"
)

const (
	topicSnapshotPrefix = "matching_snapshot_"
)

type RedisSnapshotStore struct {
	productId   string
	redisClient *redis.Client
}

func NewRedisSnapshotStore(productId string) SnapshotStore {
	gbeConfig := conf.GetConfig()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     gbeConfig.Redis.Addr,
		Password: gbeConfig.Redis.Password,
		DB:       0,
	})

	return &RedisSnapshotStore{
		productId:   productId,
		redisClient: redisClient,
	}
}

func (s *RedisSnapshotStore) Store(snapshot *Snapshot) error {
	buf, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	return s.redisClient.Set(topicSnapshotPrefix+s.productId, buf, 7*24*time.Hour).Err()
}

func (s *RedisSnapshotStore) GetLatest() (*Snapshot, error) {
	ret, err := s.redisClient.Get(topicSnapshotPrefix + s.productId).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var snapshot Snapshot
	err = json.Unmarshal(ret, &snapshot)
	return &snapshot, err
}
