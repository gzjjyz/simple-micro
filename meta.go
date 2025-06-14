package micro

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gzjjyz/confloader"
	"github.com/gzjjyz/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
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

type DBConnections struct {
	Redis map[string]*RedisConn `json:"redis"`
}

func (c *DBConnections) GetRedisConn(connName string) (*RedisConn, bool) {
	conn, ok := c.Redis[connName]
	return conn, ok
}

type HuaWeiObs struct {
	Endpoint   string
	Ak         string
	Sk         string
	BucketName string
	PathStyle  bool
}

type OBS struct {
	Bucket  string `json:"bucket"`
	Backup  int    `json:"backup"`
	Expires int    `json:"expires"`
}

type Cloud struct {
	Obs map[string]*OBS
}

type SdkSt struct {
	AppId     uint32 `json:"appid"`
	ApiSecret string `json:"api_secret"`
	ApiUrl    string `json:"api_url"`
}

type Meta struct {
	AppId          uint32 `json:"app_id"`
	Env            string `json:"env"`
	Etcd           `json:"etcd"`
	DBConnections  `json:"db"`
	Cloud          `json:"cloud"`
	path           string
	Sdk            SdkSt `json:"sdk"`
	HuaWeiObs      `json:"huawei_obs"`
	WfHealthUrl    string `json:"wf_health_url"`
	WfHealthApiKey string `json:"wf_health_api_key"`
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
	cfgLoader := confloader.NewLoader(cfgFilePath, meta)
	if err := cfgLoader.Load(); err != nil {
		return err
	}

	hasInitMeta = true

	meta.path = cfgFilePath

	return nil
}

func MustMeta() *Meta {
	if !hasInitMeta {
		panic("meta not init")
	}

	return meta
}

func (m *Meta) GetConfigPath() string {
	return m.path
}

func NewEtcdCliWithContext(ctx context.Context) (*clientv3.Client, error) {
	config := MustMeta()
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   config.Endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logger.LogError("err:%v", err)
		return nil, err
	}

	// etcd 可用性
	if len(config.Endpoints) == 0 {
		return nil, fmt.Errorf("not etcd configured end point list")
	}
	for _, url := range config.Endpoints {
		_, err = etcdClient.Status(ctx, url)
		if err != nil {
			logger.LogError("err:%v , url is %s", err, url)
			return nil, err
		}
	}
	return etcdClient, nil
}
