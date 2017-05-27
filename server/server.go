/*
Package zex.server defines gRPS DSL service
*/
package server

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pborman/uuid"
	"github.com/dmitryrpm/zex/proto"
	"github.com/dmitryrpm/zex/storage"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"io"
	"sync"
	"strings"
)

// Invoker function for mock
type Invoker func(ctx context.Context, method string, args, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error


// New MOCK constructor for Tests
func NewMock(invoker Invoker, DB storage.Database) *zexServerStruct {
	return &zexServerStruct{
		lockForRegger:    &sync.RWMutex{},
		RegisterServices: make(map[string]*grpc.ClientConn),

		lockForPather:    &sync.RWMutex{},
		PathToServices:   make(map[string][]string),

		Invoke:           invoker,
		DB:               DB,
	}
}


// New constructor
func New(DB storage.Database) zex.ZexServer {

	return &zexServerStruct{
		lockForRegger:    &sync.RWMutex{},
		RegisterServices: make(map[string]*grpc.ClientConn),

		lockForPather:    &sync.RWMutex{},
		PathToServices:   make(map[string][]string),

		Invoke:           grpc.Invoke,
		DB:               DB,
	}
}


// zexServerStruct structure
type zexServerStruct struct {
	lockForRegger    *sync.RWMutex
	RegisterServices map[string]*grpc.ClientConn

	lockForPather    *sync.RWMutex
	PathToServices   map[string][]string

	Invoke Invoker
	DB     storage.Database
}


// Register service interface impl
func (s *zexServerStruct) Register(ctx context.Context, service *zex.Service) (*zex.Empty, error) {
	grpclog.Printf("Start registraion service in Zex (%s, %s)", service.Name, service.Addr)
	// ---------------------
	// Do reflection request
	// ---------------------
	serviceKey := service.Addr + "/" + service.Name
	grpclog.Printf("start get all methods in registred \"%s\" service", serviceKey)

	// create connect
	conn, err := grpc.Dial(service.Addr, grpc.WithInsecure())
	if err != nil {
		grpclog.Printf("did not connect: %v", err)
		conn.Close()
		return nil, err
	}

	// get host, port
	ss := strings.Split(service.Addr, ":")
	host, _ := ss[0], ss[1]

	// get services info
	c := rpb.NewServerReflectionClient(conn)
	grpclog.Printf("do info request")
	informer, err := c.ServerReflectionInfo(ctx)
	if err != nil {
		grpclog.Printf("did not informer: %v", err)
		return nil, err
	}

	// send to informer request, and received answer
	grpclog.Printf("do reflation request for list services")
	err = informer.Send(&rpb.ServerReflectionRequest{
		Host:           host,
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{},
	})
	if err != nil {
		grpclog.Printf("can't send info req: %v", err)
		return nil, err

	}

	info, err := informer.Recv()
	if err != nil {
		grpclog.Printf("Recv err: %v", err)
		return nil, err
	}
	grpclog.Printf("get info: host: \"%s\"", info.String())

	// Load FileDescriptorProto with services
	for _, srv := range info.GetListServicesResponse().GetService() {
		if srv.Name != "grpc.reflection.v1alpha.ServerReflection" {
			err = informer.Send(&rpb.ServerReflectionRequest{
				Host: host,
				MessageRequest: &rpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: srv.Name},
			})
			if err != nil {
				return nil, err
			}

			answer, err := informer.Recv()
			if err != nil {
				return nil, err
			}

			// unpack file get file description
			desc, err := extractFile(answer.GetFileDescriptorResponse().FileDescriptorProto[0])
			if err != nil {
				return nil, err
			}

			// read file, get services and register it
			for _, serviceObj := range desc.GetService() {
				for _, methodObj := range serviceObj.GetMethod() {
					method := fmt.Sprintf("/%s.%s/%s", desc.GetPackage(), serviceObj.GetName(), methodObj.GetName())
					grpclog.Printf("registry method %s => %s", serviceKey, method)
					s.lockForPather.Lock()
					s.PathToServices[method] = []string{serviceKey}
					s.lockForPather.Unlock()
				}
			}

		}
	}

	// Add to RegisterServices with locks
	s.lockForRegger.Lock()
	s.RegisterServices[serviceKey] = conn
	s.lockForRegger.Unlock()

	grpclog.Printf("Add service to map with key %s successed", serviceKey)
	return &zex.Empty{}, nil
}


// Pipeline service interface impl
func (s *zexServerStruct) Pipeline(stream zex.Zex_PipelineServer) error {
	grpclog.Printf("listen stream pipeline")

	pid := uuid.New()

	pipeline := make([]*zex.Cmd, 0)

	// open stream, after close run pipeline
	transation := s.DB.NewTransaction()
	for {
		cmd, err := stream.Recv()

		if err == io.EOF {
			// send pid to client
			err = stream.SendAndClose(&zex.Pid{ID: pid})
			if err != nil {
				return err
			}

			// set transation atomic
			err = transation.Commit()

			if err != nil {
				grpclog.Printf("can't set value to storage_leveldb %s: %v", pipeline, err)
				return nil
			}

			go s.runPipeline(pid)
			return nil
		}

		if err != nil {
			grpclog.Printf("%v.Pipeline(_) = _, %v", cmd, err)
			return err
		} else {
			grpclog.Println("Add cmd to pipeline", cmd)
			transation.Put([]byte(pid + "_" +  cmd.Path), cmd.Body)
			pipeline = append(pipeline, cmd)
		}
	}

	return nil
}

// Subscribe service interface impl
func (s *zexServerStruct) Subscribe(ctx context.Context, pid *zex.Pid) (*zex.Empty, error) {
	grpclog.Printf("Start Subscribe uuid %s", pid.ID)

	return &zex.Empty{}, nil
}

// Run pipeline
func (s *zexServerStruct) runPipeline(pid string) {
	grpclog.Printf("Start RunPipeline uuid %s, storage_leveldb count %d", pid, s.DB.GetRowsCount())

	// Get context for cancel all goroutine calls
	var (
		ctx, cancel = context.WithCancel(context.Background())
		pipeline    []*zex.Cmd
	)

	// create batch transaction
	transation := s.DB.NewTransaction()

	// iterate to storage_leveldb
	iter := s.DB.GetIterator()
	for iter.Next() {
		key := iter.Key()
		if strings.Contains(string(key), pid) {
			value := iter.Value()
			ss := strings.Split(string(key), "_")
			transation.Delete(key)
			// update pipepile
			pipeline = append(pipeline, &zex.Cmd{zex.CmdType(1), ss[1], value})
		}
	}
	iter.Release()

	lengthPipiline := len(pipeline)
	grpclog.Printf("find %d pipeline commands", lengthPipiline)

	if lengthPipiline == 0 {
		grpclog.Println("no find pipelines, cancel")
		return
	}
	// Create chan errC and wait doing all goroutine
	errC := make(chan error, lengthPipiline)

	// create goroutine to task
	for _, cmd := range pipeline {
		go s.callCmd(ctx, cmd, errC)
	}

	grpclog.Printf("create %d goroutines -> callCmd... wait errors or success... %s", lengthPipiline, pid)
	for err := range errC {
		lengthPipiline--

		// If one of this fail, all failed
		if err != nil {
			grpclog.Printf("command was failed by id %s: %s", pid, err)
			cancel()
			return
		}

		// Close context if ok
		if lengthPipiline == 0 {
			grpclog.Printf("all command without errors")
			close(errC)
			err := transation.Commit()
			if err != nil {
				grpclog.Printf("level db pid %s has incorrect status %s", pid, err)
				return
			}
		}
	}

	grpclog.Printf("runPipeline %s was done, storage_leveldb cleanup, all rows %d", pid, s.DB.GetRowsCount())

}

// Call command service
func (s *zexServerStruct) callCmd(ctx context.Context, cmd *zex.Cmd, errC chan error) {
	var (
		out        = &any.Any{}
		cc         *grpc.ClientConn
		serviceKey []string
		ok         bool
	)

	s.lockForPather.RLock()
	grpclog.Println(s.PathToServices, cmd.Path)

	serviceKey, err := s.PathToServices[cmd.Path]
	s.lockForPather.RUnlock()

	if !err {
		errC <- errors.New("incorrect get serviceKey, this blocked")
	}

	// if not found
	if len(serviceKey) == 0 {
		errC <- errors.New(`not found path in PathToServices serviceKey`)
		return
	}

	s.lockForRegger.RLock()
	cc, ok = s.RegisterServices[serviceKey[0]]
	s.lockForRegger.RUnlock()

	// if incorrect path
	if !ok {
		errC <- errors.New(`not found path in RegisterServices`)
		return
	}

	grpclog.Println("start ", serviceKey, cmd.Path)
	in := byteProto(cmd.Body)
	// The grpc.Invoke or Mock in test returns an error, or nil to chan
	grpclog.Println("Start wait errors or successed")
	errC <- s.Invoke(ctx, cmd.Path, &in, out, cc)
}


func extractFile(gz []byte) (*descriptor.FileDescriptorProto, error) {
	fd := new(descriptor.FileDescriptorProto)
	if err := proto.Unmarshal(gz, fd); err != nil {
		return nil, fmt.Errorf("malformed FileDescriptorProto: %v", err)
	}

	return fd, nil
}

type byteProto []byte

func (b byteProto) Marshal() ([]byte, error) {
	return b, nil
}

func (b byteProto) Reset()         {}
func (b byteProto) String() string { return string(b) }
func (b byteProto) ProtoMessage()  {}