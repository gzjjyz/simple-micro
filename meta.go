package micro

import (
	"context"
	"fmt"
	"github.com/995933447/confloader"
	"github.com/gzjjyz/logger"
	"github.com/gzjjyz/srvlib/utils"
	"go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

type Etcd struct {
	ConnectTimeoutMs int32    `json:"connect_timeout_ms"`
	Endpoints        []string `json:"endpoints"`
}

type RedisConn struct {
	Host     string `json:"host"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type RedisConnections struct {
	Redis map[string]*RedisConn `json:"redis"`
}

func (c *RedisConnections) GetRedisConn(connName string) (*RedisConn, bool) {
	conn, ok := c.Redis[connName]
	return conn, ok
}

type Meta struct {
	Etcd             `json:"etcd"`
	RedisConnections `json:"redis"`
}

var (
	meta        *Meta
	hasInitMeta bool
	initMetaMu  sync.RWMutex
)

func InitMeta(cfgFilePath string) error {
	if cfgFilePath == "" {
		cfgFilePath = defaultCfgFilePath
	}

	if hasInitMeta {
		return nil
	}

	initMetaMu.Lock()
	defer initMetaMu.Unlock()

	if hasInitMeta {
		return nil
	}

	meta = &Meta{}
	cfgLoader := confloader.NewLoader(cfgFilePath, 5*time.Second, meta)
	if err := cfgLoader.Load(); err != nil {
		return err
	}

	hasInitMeta = true

	watchMetaErrCh := make(chan error)
	utils.ProtectGo(func() {
		cfgLoader.WatchToLoad(watchMetaErrCh)
	})
	utils.ProtectGo(func() {
		for {
			err := <-watchMetaErrCh
			if err != nil {
				utils.SafeLogErr(err, true)
			}
		}
	})

	return nil
}

func MustMeta() *Meta {
	if !hasInitMeta {
		panic("meta not init")
	}

	return meta
}

func NewEtcdCliWithContext(ctx context.Context) (*clientv3.Client, error) {
	config := MustMeta()
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logger.Errorf("err:%v", err)
		return nil, err
	}

	// etcd 可用性
	if len(config.Endpoints) == 0 {
		return nil, fmt.Errorf("not etcd configured end point list")
	}
	for _, url := range config.Endpoints {
		_, err = etcdClient.Status(ctx, url)
		if err != nil {
			logger.Errorf("err:%v , url is %s", err, url)
			return nil, err
		}
	}
	return etcdClient, nil
}
