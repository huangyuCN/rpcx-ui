package main

import "testing"

func TestFetchServices(t *testing.T) {
	etcdv3 := &EtcdV3Registry{}
	etcdv3.initRegistry([]string{"127.0.0.1:2379"}, "com/gh")
	svrs := etcdv3.fetchServices()
	t.Logf("svrs:%v", svrs)
}
