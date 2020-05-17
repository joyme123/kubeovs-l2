// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package etcd

import (
	"context"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
)

const (
	EtcdLockKey = "/kubeovsio/ipam"
)

// EtcdLock use etcd
type EtcdLock struct {
	mutex *concurrency.Mutex
	sess  *concurrency.Session
}

// NewEtcdLock use etcd as lock
func NewEtcdLock(cli *clientv3.Client) (*EtcdLock, error) {
	sess, err := concurrency.NewSession(cli)
	if err != nil {
		return nil, err
	}

	m := concurrency.NewMutex(sess, EtcdLockKey)
	return &EtcdLock{mutex: m}, nil
}

func (l *EtcdLock) Close() error {
	return l.sess.Close()
}

// Lock acquires an exclusive lock
func (l *EtcdLock) Lock() error {
	return l.mutex.Lock(context.TODO())
}

// Unlock releases the lock
func (l *EtcdLock) Unlock() error {
	return l.mutex.Unlock(context.TODO())
}
