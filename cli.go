package micro

import (
	"context"
	"github.com/gzjjyz/logger"
	"google.golang.org/grpc"
)

type InitGrpcClientFunc func(conn *grpc.ClientConn)

type Cli struct {
	name               string
	target             string
	etcdDiscover       bool
	initGrpcClientFunc InitGrpcClientFunc
}

func NewCli(name string, opts ...CliOption) *Cli {
	cli := &Cli{name: name}
	for _, opt := range opts {
		opt(cli)
	}
	return cli
}

func (cli *Cli) defaultDialOptionList() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
	}
}

func (cli *Cli) DialContext(ctx context.Context) error {
	target := cli.target
	var optList []grpc.DialOption

	// 使用服务发现
	if cli.etcdDiscover {
		dialOptions, etcdTarget, err := discoverToEtcd(cli.name)
		if err != nil {
			logger.Errorf("err:%v", err)
			return err
		}
		optList = append(optList, dialOptions...)
		target = etcdTarget
	}

	// 创建 grpc 连接代理
	optList = append(optList, cli.defaultDialOptionList()...)
	conn, err := grpc.DialContext(
		ctx,
		target,
		optList...,
	)
	if err != nil {
		logger.Errorf("dial %s failed , etcd target is %s , err:%v", cli.name, target, err)
		return err
	}

	if cli.initGrpcClientFunc != nil {
		cli.initGrpcClientFunc(conn)
	}

	return nil
}

type CliOption func(*Cli)

// WithCliEtcdDiscover etcd 服务发现
func WithCliEtcdDiscover() CliOption {
	return func(cli *Cli) {
		cli.etcdDiscover = true
	}
}

// WithCliTarget 不使用代理 直连
func WithCliTarget(t string) CliOption {
	return func(cli *Cli) {
		cli.target = t
	}
}

func WithInitGrpcClientFunc(f InitGrpcClientFunc) CliOption {
	return func(cli *Cli) {
		cli.initGrpcClientFunc = f
	}
}
