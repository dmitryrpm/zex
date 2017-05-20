package main

import (
	"io"
	"net"
	"fmt"
	"flag"
	"github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "github.com/zex/zex/proto"
	"google.golang.org/grpc/grpclog"
)

var (
	port = flag.Int("port", 10000, "The server port")
)

type ZexServer struct {
	RegisterServices   map[string]*pb.Service
	PipelineInfo map[string][]*pb.Cmd
}

// Register service interface impl
func (s *ZexServer) Register(ctx context.Context, service *pb.Service) (*pb.Empty, error) {
	grpclog.Printf("Start registraion service in Zex (%s, %s)", service.Name, service.Addr)
	s.RegisterServices[service.Name] = service
	grpclog.Printf("Add service to map with key %s successed", s.RegisterServices)
	return &pb.Empty{}, nil
}

// Pipeline service interface impl
func (s *ZexServer) Pipeline(stream pb.Zex_PipelineServer) (error)  {
	grpclog.Printf("listen stream pipeline")

	uuid4 := uuid.NewV4().String()

	s.PipelineInfo[string(uuid4)] = make([]*pb.Cmd, 0, 5)

	for {
		cmd, err := stream.Recv()
		if err == io.EOF {
			grpclog.Printf("close stream, %s", s.PipelineInfo)
			stream.SendAndClose(&pb.Pid{string(uuid4)})
			return nil
		}

		if err != nil {
			grpclog.Fatalf("%v.Pipeline(_) = _, %v", cmd, err)
		} else {
			grpclog.Println("Add cmd to pipeline", cmd)
			s.PipelineInfo[string(uuid4)] = append(s.PipelineInfo[string(uuid4)], cmd)
		}
	}

	return nil
}

// Subscribe service interface impl
func (s *ZexServer) Subscribe (ctx context.Context, pid *pb.Pid) (*pb.Empty, error) {
	grpclog.Printf("Start Subscribe uuid %s", pid.ID)



	return &pb.Empty{}, nil
}

// Run pipeline service interface
func (s *ZexServer) RunPipeline (ctx context.Context, pid *pb.Pid) (*pb.Empty, error) {
	grpclog.Printf("Start RunPipeline uuid %s", pid)
	delete(s.PipelineInfo, string(pid.ID))
	grpclog.Printf("PipelineInfo, %s", s.PipelineInfo)
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
	zs.PipelineInfo = make(map[string][]*pb.Cmd)
	pb.RegisterZexServer(grpcServer, zs)
	grpcServer.Serve(tcp_connect)
}