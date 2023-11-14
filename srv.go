package micro

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/gzjjyz/logger"
	"github.com/gzjjyz/srvlib/utils"
	"google.golang.org/grpc"
)

type Srv struct {
	name      string
	ipVar     string
	port      int
	pprofPort int

	registerCustomServiceServer func(*grpc.Server)
	regEtcd                     bool
}

func NewSrv(name string, ipVar string, port int, opts ...SrvOption) *Srv {
	s := &Srv{name: name, ipVar: ipVar, port: port}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type SrvOption func(srv *Srv)

func WithSrvRegisterServerFunc(f func(*grpc.Server)) SrvOption {
	return func(srv *Srv) {
		srv.registerCustomServiceServer = f
	}
}

func WithSrvPprofPort(pprofPort int) SrvOption {
	return func(srv *Srv) {
		srv.pprofPort = pprofPort
	}
}

func WithSrvRegEtcd() SrvOption {
	return func(srv *Srv) {
		srv.regEtcd = true
	}
}

func (srv *Srv) Serve(ctx context.Context) error {
	// pprof 监控
	if srv.pprofPort > 0 {
		utils.ProtectGo(func() {
			err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", srv.pprofPort), nil)
			if err != nil {
				logger.LogError(err.Error())
			}
		})
	}

	ip, err := utils.EvalVarToParseIp(srv.ipVar)
	if err != nil {
		logger.LogError(err.Error())
		return err
	}

	grpcSrv := grpc.NewServer()
	if srv.registerCustomServiceServer != nil {
		srv.registerCustomServiceServer(grpcSrv)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, srv.port))
	if err != nil {
		return err
	}

	// 服务注册 -> etcd
	if srv.regEtcd {
		err = RegisterToEtcd(ctx, fmt.Sprintf("%s:%d", ip, srv.port), srv.name)
		if err != nil {
			logger.LogError("err:%v", err)
			return err
		}
	}

	// 启动
	err = grpcSrv.Serve(listener)
	if err != nil {
		return err
	}
	if err != nil {
		logger.LogError(err.Error())
		return err
	}
	return nil
}
