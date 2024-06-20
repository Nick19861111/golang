package discovery

import (
	"common/config"
	"common/logs"
	"context"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/resolver"
	"time"
)

type Resolver struct {
	conf        config.EtcdConf
	etcdCl      *clientv3.Client
	DialTimeout int
	closeCh     chan struct{}
	key         string
	cc          resolver.ClientConn
	srvAddrList []resolver.Address
	watchCh     clientv3.WatchChan
}

func (r Resolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	//TODO implement me
	r.cc = cc
	//链接etcd
	var err error
	r.etcdCl, err = clientv3.New(clientv3.Config{
		Endpoints:   r.conf.Addrs,
		DialTimeout: time.Duration(r.conf.DialTimeout) * time.Second,
	})
	if err != nil {
		logs.Fatal("create etcd resolver client error:", err)
	}
	r.closeCh = make(chan struct{})
	//2.根据key或者value
	r.key = target.URL.Path
	if err := r.sync(); err != nil {
		return nil, err
	}

	//节点变动的时候操作
	go r.watch()
	return nil, nil
}

// 同步
func (r Resolver) sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.conf.RWTimeout)*time.Second)
	defer cancel()
	res, err := r.etcdCl.Get(ctx, r.key, clientv3.WithPrefix())
	if err != nil {
		logs.Error("get etcd resolver err:", err)
		return err
	}
	r.srvAddrList = []resolver.Address{}
	for _, v := range res.Kvs {
		server, err := ParseValue(v.Value)
		if err != nil {
			logs.Error("parse value err:", err)
			continue
		}
		//构建地址
		r.srvAddrList = append(r.srvAddrList, resolver.Address{
			Addr:       server.Addr,
			Attributes: attributes.New("weight", server.Weight),
		})
	}

	if len(r.srvAddrList) == 0 {
		return nil
	}

	err = r.cc.UpdateState(resolver.State{
		Addresses: r.srvAddrList,
	})
	if err != nil {
		logs.Error("update state err:", err)
	}
	return nil
}

func (r Resolver) Scheme() string {
	//TODO implement me
	return "etcd"
}

func (r Resolver) watch() {

	r.watchCh = r.etcdCl.Watch(context.Background(), r.key, clientv3.WithPrefix())

	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-r.closeCh:
			r.Close()
		case res, ok := <-r.watchCh:
			if !ok {
				r.update(res.Events)
			}
		case <-ticker.C:
			if err := r.sync(); err != nil {
				logs.Fatal("watch sync err:", err)
			}
		}
	}
}

func (r Resolver) update(events []*clientv3.Event) {
	for _, ev := range events {
		switch ev.Type {
		case clientv3.EventTypePut:
			server, err := ParseValue(ev.Kv.Value)
			if err != nil {
				logs.Error("EventTypePut parse value err:", err)
			}
			addr := resolver.Address{
				Addr:       server.Addr,
				Attributes: attributes.New("weight", server.Weight),
			}
			if !Exist(r.srvAddrList, addr) {
				r.srvAddrList = append(r.srvAddrList, addr)
				err = r.cc.UpdateState(resolver.State{
					Addresses: r.srvAddrList,
				})
				if err != nil {
					logs.Error("grpc client update() failed name,err")
				}
			}
		case clientv3.EventTypeDelete:
			//删除操作
			server, err := ParseKey(string(ev.Kv.Key))
			if err != nil {
				logs.Error("EventTypeDelete parse value err:", err)
			}
			addr := resolver.Address{Addr: server.Addr}
			//r.srvAddrList
			if list, ok := Remove(r.srvAddrList, addr); ok {
				r.srvAddrList = list
				err = r.cc.UpdateState(resolver.State{
					Addresses: r.srvAddrList,
				})
				if err != nil {
					logs.Error("update state err:", err)
				}
			}
		}
	}
}

func (r Resolver) Close() {
	if r.etcdCl != nil {
		err := r.etcdCl.Close()
		if err != nil {
			logs.Fatal("etcdClClo Close is err:", err)
		}
	}
}

func Exist(list []resolver.Address, addr resolver.Address) bool {
	for i := range list {
		if list[i].Addr == addr.Addr {
			return true
		}
	}
	return false
}

func Remove(list []resolver.Address, addr resolver.Address) ([]resolver.Address, bool) {
	for i := range list {
		if list[i].Addr == addr.Addr {
			list[i] = list[len(list)-1]
			return list[:len(list)-1], true
		}
	}
	return nil, false
}

func NewResolver(conf config.EtcdConf) *Resolver {
	return &Resolver{
		conf:        conf,
		DialTimeout: conf.DialTimeout,
	}
}
