package micro

import (
	"fmt"
	"github.com/gzjjyz/logger"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"time"
)

func discoverToEtcd(serverName string) ([]grpc.DialOption, string, error) {
	var optList []grpc.DialOption

	etcdClient, err := NewEtcdCliWithContext(WithGCtxTimeout(time.Second * 5))
	if err != nil {
		logger.Errorf("err:%v", err)
		return nil, "", err
	}

	target := fmt.Sprintf("etcd:///%s", serverName)
	etcdResolverBuilder, err := resolver.NewBuilder(etcdClient)
	if err != nil {
		logger.Errorf("err:%v", err)
		return nil, "", err
	}

	optList = append(optList,
		grpc.WithResolvers(etcdResolverBuilder),                                                      // 注入 etcd bresolver
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, roundrobin.Name)), // 声明使用的负载均衡策略为 round robin
	)

	return optList, target, err
}
