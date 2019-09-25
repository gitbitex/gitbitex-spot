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
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	logger "github.com/siddontang/go-log/log"
)

type KafkaLogReader struct {
	readerId  string
	productId string
	reader    *kafka.Reader
	observer  LogObserver
}

func NewKafkaLogReader(readerId, productId string, brokers []string) LogReader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   brokers,
		Topic:     topicBookMessagePrefix + productId,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10e6,
	})
	return &KafkaLogReader{readerId: readerId, productId: productId, reader: reader}
}

func (r *KafkaLogReader) GetProductId() string {
	return r.productId
}

func (r *KafkaLogReader) RegisterObserver(observer LogObserver) {
	r.observer = observer
}

func (r *KafkaLogReader) Run(seq, offset int64) {
	logger.Infof("%v:%v read from %v", r.productId, r.readerId, offset)

	var lastSeq = seq

	err := r.reader.SetOffset(offset)
	if err != nil {
		panic(err)
	}

	for {
		kMessage, err := r.reader.FetchMessage(context.Background())
		if err != nil {
			logger.Error(err)
			continue
		}

		var base Base
		err = json.Unmarshal(kMessage.Value, &base)
		if err != nil {
			panic(err)
		}

		if base.Sequence <= lastSeq {
			// 丢弃重复的log
			logger.Infof("%v:%v discard log :%+v", r.productId, r.readerId, base)
			continue
		} else if lastSeq > 0 && base.Sequence != lastSeq+1 {
			// seq发生不连续，可能是撮合引擎发生了严重错误
			logger.Fatalf("non-sequence detected, lastSeq=%v seq=%v", lastSeq, base.Sequence)
		}
		lastSeq = base.Sequence

		switch base.Type {
		case LogTypeOpen:
			var log OpenLog
			err := json.Unmarshal(kMessage.Value, &log)
			if err != nil {
				panic(err)
			}
			r.observer.OnOpenLog(&log, kMessage.Offset)

		case LogTypeMatch:
			var log MatchLog
			err := json.Unmarshal(kMessage.Value, &log)
			if err != nil {
				panic(err)
			}
			r.observer.OnMatchLog(&log, kMessage.Offset)

		case LogTypeDone:
			var log DoneLog
			err := json.Unmarshal(kMessage.Value, &log)
			if err != nil {
				panic(err)
			}
			r.observer.OnDoneLog(&log, kMessage.Offset)

		}
	}
}
