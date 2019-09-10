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
	"sync"
)

type subscription struct {
	subscribers map[string]map[int64]*Client
	mu          sync.RWMutex
}

func newSubscription() *subscription {
	return &subscription{subscribers: map[string]map[int64]*Client{}}
}

func (s *subscription) subscribe(channel string, client *Client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, found := s.subscribers[channel]
	if !found {
		s.subscribers[channel] = map[int64]*Client{}
	}

	_, found = s.subscribers[channel][client.id]
	if found {
		return false
	}
	s.subscribers[channel][client.id] = client
	return true
}

func (s *subscription) unsubscribe(channel string, client *Client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, found := s.subscribers[channel]
	if !found {
		return false
	}

	_, found = s.subscribers[channel][client.id]
	if !found {
		return false
	}
	delete(s.subscribers[channel], client.id)
	return true
}

func (s *subscription) publish(channel string, msg interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, found := s.subscribers[channel]
	if !found {
		return
	}

	for _, c := range s.subscribers[channel] {
		c.writeCh <- msg
	}
}
