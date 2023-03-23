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

package main

import (
	"github.com/gitbitex/gitbitex-spot/conf"
	"github.com/gitbitex/gitbitex-spot/matching"
	"github.com/gitbitex/gitbitex-spot/models"
	"github.com/gitbitex/gitbitex-spot/pushing"
	"github.com/gitbitex/gitbitex-spot/rest"
	"github.com/gitbitex/gitbitex-spot/service"
	"github.com/gitbitex/gitbitex-spot/worker"
	"github.com/prometheus/common/log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	gbeConfig := conf.GetConfig()

	go func() {
		log.Info(http.ListenAndServe("localhost:6060", nil))
	}()

	go models.NewBinLogStream().Start()

	matching.StartEngine()

	pushing.StartServer()

	worker.NewFillExecutor().Start()
	worker.NewBillExecutor().Start()
	products, err := service.GetProducts()
	if err != nil {
		panic(err)
	}
	for _, product := range products {
		worker.NewTickMaker(product.Id, matching.NewKafkaLogReader("tickMaker", product.Id, gbeConfig.Kafka.Brokers)).Start()
		worker.NewFillMaker(matching.NewKafkaLogReader("fillMaker", product.Id, gbeConfig.Kafka.Brokers)).Start()
		worker.NewTradeMaker(matching.NewKafkaLogReader("tradeMaker", product.Id, gbeConfig.Kafka.Brokers)).Start()
	}

	rest.StartServer()

	select {}
}
