package main

import (
	"flag"
	"net"
	"fmt"
	"io"
	_ "os/exec"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "github.com/zex/zex/proto"
	_ "github.com/golang/protobuf/proto"
	"google.golang.org/grpc/grpclog"

	_ "text/template/parse"
)

var (
	port = flag.Int("port", 10000, "The server port")
)

type ZexServer struct {
	RegisterServices   map[string]*pb.Service
}

// Register service interface impl
func (s *ZexServer) Register(ctx context.Context, service *pb.Service) (*pb.Empty, error) {
	grpclog.Printf("Start registraion service in Zex (%s, %s)", service.Name, service.Addr)
	s.RegisterServices[service.Name] = service
	grpclog.Printf("Add service to map with key %s successed", s.RegisterServices)
	return &pb.Empty{}, nil
}

// Pipeline service interface impl
func (s *ZexServer) Pipeline(stream pb.Zex_PipelineServer) (error) { //(*pb.Pid, error) {
	grpclog.Printf("listen stream pipeline")

	//gen_uuid, err := exec.Command("uuidgen").Output()
	//if err != nil {
	//	grpclog.Fatal(err)
	//}
	//uuid := pb.Pid{string(gen_uuid)}

	for {
		cmd, err := stream.Recv()
		if err == io.EOF {
			grpclog.Printf("close stream")
			//return &pb.Pid{"123"}, nil
			//return &pb.Empty{}, nil
			return nil
		}

		if err != nil {
			grpclog.Fatalf("%v.Pipeline(_) = _, %v", cmd, err)
		} else {
			grpclog.Println("Add cmd to pipeline", cmd)
		}
	}

	//return &pb.Pid{"123"}, nil
	//return &pb.Empty{}, nil
	return nil
}

// Subscribe service interface impl
func (s *ZexServer) Subscribe (ctx context.Context, pid *pb.Pid) (*pb.Empty, error) {
	grpclog.Printf("Check pipeline pid %s", pid.ID)
	return &pb.Empty{}, nil
}

// Run pipeline service interface
func (s *ZexServer) RunPipeline (ctx context.Context, pid *pb.Pid) (*pb.Empty, error) {
	grpclog.Printf("Start pipleline pid %s")
	return &pb.Empty{}, nil
}

func main() {
	flag.Parse()
	tcp_connect, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	zs := new(ZexServer)
	zs.RegisterServices = make(map[string]*pb.Service)
	pb.RegisterZexServer(grpcServer, zs)
	grpcServer.Serve(tcp_connect)
}