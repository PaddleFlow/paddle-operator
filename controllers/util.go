// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"sync"
	"time"
)

type Cache struct {
	Data map[string]bool

	mx *sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		Data: map[string]bool{},
		mx:   &sync.RWMutex{},
	}
}

func (c *Cache) Get(key string) (ok bool) {
	c.mx.RLock()
	_, ok = c.Data[key]
	c.mx.RUnlock()
	return ok
}

func (c *Cache) Add(key string, ttl int) {
	if ttl <= 0 {
		return
	}
	c.mx.Lock()
	c.Data[key] = true
	c.mx.Unlock()

	time.AfterFunc(time.Duration(ttl)*time.Second, func() { c.del(key) })
}

func (c *Cache) del(key string) {
	c.mx.Lock()
	delete(c.Data, key)
	c.mx.Unlock()
}
