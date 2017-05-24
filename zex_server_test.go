package main_test

import (
	"testing"
	zex "zex/server"
	zex_proto "zex/proto"
	"golang.org/x/net/context"
	//"google.golang.org/grpc"
	"sync"
	"google.golang.org/grpc/grpclog"
)


type MockZexServer struct{
	zex.ZexServer
	lockForRegger    *sync.RWMutex
	lockForPipper    *sync.RWMutex
	lockForPather    *sync.RWMutex
}


func (s *MockZexServer) runCmd(ctx context.Context, cmd *zex_proto.Cmd, errC chan error) {
	grpclog.Println("OLOLOLO")
	errC <- nil
}


//func mockNew() MockZexServer {
//	zs := MockZexServer{
//		lockForRegger:    &sync.RWMutex{},
//		lockForPipper:    &sync.RWMutex{},
//		lockForPather:    &sync.RWMutex{},
//	}
//	return zs
//}

//func (s *MockZexServer) Pipeline(stream zex_proto.Zex_PipelineServer) error {
//	return  nil
//}
//
//
//func TestZexServerPipeline(t *testing.T) {
//	zs := MockZexServer{}
//	_, err := zs.Pipeline()
//	if err != nil {
//		t.Error("Pipeline err expected nil, got ", err)
//	}
//}


func TestZexServerPipeline(t *testing.T) {
	zs := MockZexServer{}
	zs.PipelineInfo = make(map[string][]*zex_proto.Cmd)
	zs.lockForPipper = &sync.RWMutex{}

	cmd := zex_proto.Cmd{1, "/A/A/A", []byte("seafood")}
	pipeline := make([]*zex_proto.Cmd, 0)
	pipeline = append(pipeline, &cmd)
	zs.PipelineInfo["pid"] = pipeline

	// call rpc function
	grpclog.Println("before PipelineInfo=",  zs.PipelineInfo)
	zs.RunPipeline("pid")
	grpclog.Println("after PipelineInfo=",  zs.PipelineInfo)
}


//func TestZexServerRegister(t *testing.T) {
//	zs := mockNew()
//	cx := context.Background()
//	service := zex_proto.Service{"Name", "Addr"}
//	_, err := zs.Register(cx, &service)
//	if err != nil {
//		t.Error("Register err expected nil, got ", err)
//	}
//}