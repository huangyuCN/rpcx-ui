package main

import (
	"encoding/base64"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/docker/libkv"
	kvstore "github.com/docker/libkv/store"
	etcd "github.com/smallnest/libkv-etcdv3-store"
)

// Service is a service endpoint
type Service struct {
	ID       string
	Name     string
	Address  string
	Metadata string
	State    string
	Group    string
}

type EtcdV3Registry struct {
	kv             kvstore.Store
	serviceBaseUrl string
}

func (r *EtcdV3Registry) initRegistry(etcdUrls []string, serviceBaseUrl string) {
	etcd.Register()

	kv, err := libkv.NewStore(etcd.ETCDV3, etcdUrls, nil)
	if err != nil {
		log.Printf("cannot create etcd registry: %v", err)
		return
	}
	r.kv = kv
	r.serviceBaseUrl = serviceBaseUrl

	return
}

func (r *EtcdV3Registry) fetchServices() []*Service {
	var services []*Service
	kvs, err := r.kv.List(r.serviceBaseUrl)
	if err != nil {
		log.Printf("failed to list services %s: %v", r.serviceBaseUrl, err)
		return services
	}

	for _, value := range kvs {

		nodes, err := r.kv.List(value.Key)
		if err != nil {
			log.Printf("failed to list %s: %v", value.Key, err)
			continue
		}

		for _, n := range nodes {
			key := string(n.Key[:])
			i := strings.LastIndex(key, "/")
			serviceName := strings.TrimPrefix(key[0:i], r.serviceBaseUrl)
			var serviceAddr string
			fields := strings.Split(key, "/")
			if fields != nil && len(fields) > 1 {
				serviceAddr = fields[len(fields)-1]
			}
			v, err := url.ParseQuery(string(n.Value[:]))
			if err != nil {
				log.Println("etcd value parse failed. error: ", err.Error())
				continue
			}
			state := "n/a"
			group := ""
			if err == nil {
				state = v.Get("state")
				if state == "" {
					state = "active"
				}
				group = v.Get("group")
			}
			id := base64.StdEncoding.EncodeToString([]byte(serviceName + "@" + serviceAddr))
			service := &Service{ID: id, Name: serviceName, Address: serviceAddr, Metadata: string(n.Value[:]), State: state, Group: group}
			services = append(services, service)
		}

	}

	return services
}

func (r *EtcdV3Registry) deactivateService(name, address string) error {
	key := path.Join(r.serviceBaseUrl, name, address)

	kv, err := r.kv.Get(key)

	if err != nil {
		return err
	}

	v, err := url.ParseQuery(string(kv.Value[:]))
	if err != nil {
		log.Println("etcd value parse failed. err ", err.Error())
		return err
	}
	v.Set("state", "inactive")
	err = r.kv.Put(kv.Key, []byte(v.Encode()), &kvstore.WriteOptions{IsDir: false})
	if err != nil {
		log.Println("etcd set failed, err : ", err.Error())
	}

	return err
}

func (r *EtcdV3Registry) activateService(name, address string) error {
	key := path.Join(r.serviceBaseUrl, name, address)
	kv, err := r.kv.Get(key)

	v, err := url.ParseQuery(string(kv.Value[:]))
	if err != nil {
		log.Println("etcd value parse failed. err ", err.Error())
		return err
	}
	v.Set("state", "active")
	err = r.kv.Put(kv.Key, []byte(v.Encode()), &kvstore.WriteOptions{IsDir: false})
	if err != nil {
		log.Println("etcdv3 put failed. err: ", err.Error())
	}

	return err
}

func (r *EtcdV3Registry) updateMetadata(name, address string, metadata string) error {
	key := path.Join(r.serviceBaseUrl, name, address)
	err := r.kv.Put(key, []byte(metadata), &kvstore.WriteOptions{IsDir: false})
	return err
}
