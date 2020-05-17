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
	"errors"
	"net"
	"strings"

	"github.com/containernetworking/plugins/plugins/ipam/host-local/backend"
	"go.etcd.io/etcd/clientv3"
)

const lastIPFilePrefix = "last_reserved_ip."
const LineBreak = "\r\n"

var defaultKeyPrefix = "/kubeovsio/networks/"
var (
	ErrKeyExists      = errors.New("key already exists")
	ErrKeyNotExists   = errors.New("key not exists")
	ErrWaitMismatch   = errors.New("unexpected wait result")
	ErrTooManyClients = errors.New("too many clients")
	ErrNoWatcher      = errors.New("no watcher channel")
)

// Store is a simple etcd-backed store that creates one key per IP
// address in a given directory. The contents of the key are the container ID.
type Store struct {
	*EtcdLock
	keyPrefix string
	cli       *clientv3.Client
}

// Store implements the Store interface
var _ backend.Store = &Store{}

func New(conf clientv3.Config, network, key string) (*Store, error) {
	if key == "" {
		key = defaultKeyPrefix
	}
	netWorkKey := key + network

	cli, err := clientv3.New(conf)
	if err != nil {
		return nil, err
	}

	lk, err := NewEtcdLock(cli)
	if err != nil {
		return nil, err
	}
	return &Store{lk, netWorkKey, cli}, nil
}

func (s *Store) Reserve(containerID string, ifname string, ip net.IP, rangeID string) (bool, error) {
	key := s.keyPrefix + "/" + ip.String()

	_, err := s.get(key)
	if err != nil && err != ErrKeyNotExists {
		return false, err
	}
	if err == nil {
		return false, nil
	}

	if err := s.set(key, strings.TrimSpace(containerID)+LineBreak+ifname); err != nil {
		return false, err
	}

	// store the reserved ip in lastIPFile
	ipKey := s.keyPrefix + "/" + lastIPFilePrefix + rangeID
	err = s.set(ipKey, ip.String())
	if err != nil {
		return false, err
	}
	return true, nil
}

// LastReservedIP returns the last reserved IP if exists
func (s *Store) LastReservedIP(rangeID string) (net.IP, error) {
	ipKey := s.keyPrefix + "/" + lastIPFilePrefix + rangeID
	data, err := s.get(ipKey)
	if err != nil {
		return nil, err
	}
	return net.ParseIP(data), nil
}

func (s *Store) Release(ip net.IP) error {
	return s.delete(s.keyPrefix + "/" + ip.String())
}

func (s *Store) FindByKey(containerID string, ifname string, match string) (bool, error) {
	strs, err := s.list(s.keyPrefix)
	if err != nil {
		return false, err
	}

	for _, s := range strs {
		if s == match {
			return true, nil
		}
	}

	return false, nil
}

func (s *Store) FindByID(id string, ifname string) bool {
	s.Lock()
	defer s.Unlock()

	found := false
	match := strings.TrimSpace(id) + LineBreak + ifname
	found, err := s.FindByKey(id, ifname, match)

	// Match anything created by this id
	if !found && err == nil {
		match := strings.TrimSpace(id)
		found, err = s.FindByKey(id, ifname, match)
	}

	return found
}

func (s *Store) ReleaseByKey(id string, ifname string, match string) (bool, error) {
	strs, err := s.list(s.keyPrefix)
	if err != nil {
		return false, err
	}
	for k, v := range strs {
		if strings.TrimSpace(string(v)) == match {
			err := s.delete(k)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

// N.B. This function eats errors to be tolerant and
// release as much as possible
func (s *Store) ReleaseByID(id string, ifname string) error {
	found := false
	match := strings.TrimSpace(id) + LineBreak + ifname
	found, err := s.ReleaseByKey(id, ifname, match)

	// For backwards compatibility, look for files written by a previous version
	if !found && err == nil {
		match := strings.TrimSpace(id)
		found, err = s.ReleaseByKey(id, ifname, match)
	}
	return err
}

// GetByID returns the IPs which have been allocated to the specific ID
func (s *Store) GetByID(id string, ifname string) []net.IP {
	var ips []net.IP

	match := strings.TrimSpace(id) + LineBreak + ifname
	// matchOld for backwards compatibility
	matchOld := strings.TrimSpace(id)

	strs, err := s.list(s.keyPrefix)
	if err != nil {
		return ips
	}

	for ipStr, data := range strs {
		if strings.TrimSpace(data) == match || strings.TrimSpace(data) == matchOld {
			if ip := net.ParseIP(ipStr); ip != nil {
				ips = append(ips, ip)
			}
		}
	}

	return ips
}

func (s *Store) set(key string, value string) error {

	_, err := s.cli.KV.Put(context.TODO(), key, value)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) get(key string) (string, error) {
	resp, err := s.cli.KV.Get(context.TODO(), key)
	if err != nil {
		return "", err
	}

	if resp.Count == 0 {
		return "", ErrKeyNotExists
	}

	return string(resp.Kvs[0].Value), nil
}

func (s *Store) delete(key string) error {
	_, err := s.cli.KV.Delete(context.TODO(), key)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) list(prefix string) (map[string]string, error) {
	resp, err := s.cli.KV.Get(context.TODO(), prefix, []clientv3.OpOption{
		clientv3.WithPrefix(),
	}...)

	if err != nil {
		return nil, err
	}

	res := make(map[string]string)
	for i := 0; i < int(resp.Count); i++ {
		res[string(resp.Kvs[i].Key)] = string(resp.Kvs[i].Value)
	}

	return res, nil
}
