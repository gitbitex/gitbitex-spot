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

package pushing

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/siddontang/go-log/log"
	"io/ioutil"
	"net/http"
)

type Server struct {
	addr string
	path string
	sub  *subscription
}

func NewServer(addr, path string, sub *subscription) *Server {
	return &Server{
		addr: addr,
		path: path,
		sub:  sub,
	}
}

func (s *Server) ws(c *gin.Context) {
	upGrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error(err)
		return
	}

	NewClient(conn, s.sub).startServe()
}

func (s *Server) Run() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard

	r := gin.Default()
	r.GET(s.path, s.ws)
	err := r.Run(s.addr)
	if err != nil {
		panic(err)
	}
}
