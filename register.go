package micro

import (
	"context"
	"fmt"
	"time"

	"github.com/gzjjyz/logger"
	"github.com/gzjjyz/srvlib/utils"
	"github.com/gzjjyz/srvlib/utils/signal"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
)

func RegisterToEtcd(ctx context.Context, addr, svrName string) error {
	logger.LogInfo("register svr is %s, address: %s", svrName, addr)

	var gen5sTimeout = func() context.Context {
		return WithGCtxTimeout(time.Second * 5)
	}

	etcdClient, err := NewEtcdCliWithContext(gen5sTimeout())
	if err != nil {
		logger.LogError("err:%v", err)
		return err
	}

	etcdManager, err := endpoints.NewManager(etcdClient, svrName)
	if err != nil {
		logger.LogError("err:%v", err)
		return err
	}

	// 创建一个租约，每隔 10s 需要向 etcd 汇报一次心跳，证明当前节点仍然存活
	var ttl int64 = 10
	lease, err := etcdClient.Grant(gen5sTimeout(), ttl)
	if err != nil {
		logger.LogError("err:%v", err)
		return err
	}

	// 添加注册节点到 etcd 中，并且携带上租约 id
	var k = fmt.Sprintf("%s/%s", svrName, addr)
	err = etcdManager.AddEndpoint(gen5sTimeout(), k, endpoints.Endpoint{Addr: addr}, clientv3.WithLease(lease.ID))
	if err != nil {
		logger.LogError("err:%v", err)
		return err
	}

	logger.LogDebug("registered endpoint ok, key is %s", fmt.Sprintf("%s/%s", svrName, addr))

	utils.ProtectGo(func() {
		// 每隔 5 s进行一次延续租约的动作
		for {
			select {
			case <-time.After(5 * time.Second):
				// 续约操作
				resp, err := etcdClient.KeepAliveOnce(gen5sTimeout(), lease.ID)
				if err != nil {
					logger.LogError("err:%v", err)
					continue
				}
				logger.LogDebug("keep alive resp: %+v", resp)
			case <-signal.SignalChan():
				logger.LogInfo("sign stop EndPointToEtcd")
				err := etcdManager.DeleteEndpoint(gen5sTimeout(), k)
				if err != nil {
					logger.LogError("err:%v", err)
					return
				}
				return
			case <-ctx.Done():
				logger.LogInfo("ctx stop EndPointToEtcd")
				err := etcdManager.DeleteEndpoint(gen5sTimeout(), k)
				if err != nil {
					logger.LogError("err:%v", err)
					return
				}
				return
			}
		}
	})
	return nil
}
