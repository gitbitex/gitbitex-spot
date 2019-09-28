<p align="center"><img width="40%" src="https://getbitex.oss-cn-beijing.aliyuncs.com/projects/image/logo.svg" /></p>

[![Go Report Card](https://goreportcard.com/badge/github.com/gitbitex/gitbitex-spot)](https://goreportcard.com/report/github.com/gitbitex/gitbitex-spot)

GitBitEx is an open source cryptocurrency exchange.

## Architecture
<p align="center"><img width="100%" src="https://oooooo.oss-cn-hangzhou.aliyuncs.com/gitbitex.png?v=1" /></p>

## Demo
https://gitbitex.com:8080/trade/BTC-USDT

## Dependencies
* MySql (**BINLOG[ROW format]** enabled)
* Kafka
* Redis

## Install
* Create database and make sure **BINLOG[ROW format]** enabled
* Execute ddl.sql
* Modify conf.json
* Run go build
* Run ./gitbitex-spot

## Problems?
Please let me know if you have any problems. You can submit an issue or send me an email (greensheng2001@gmail.com)