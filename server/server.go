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
	"time"
)

// Invoker function for mock
type Invoker func(ctx context.Context, method string, args, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error
type Dialer func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)

type zexOptions func(*zexServer)

func WithMockerParams(invoker Invoker, dialer Dialer) zexOptions {
	return func(srv *zexServer) {
		srv.Invoke = invoker
		srv.Dial = dialer
	}
}


const defaultTimeout = 3 * time.Second
const defaultLoopTimeout = 200 * time.Microsecond
const statusPrefix = "ST_"


// New constructor
func New(DB storage.Database, opts ...zexOptions) *zexServer {

	srv := &zexServer{
		// Services connection maps A => [connect]
		lockForRegger:    &sync.RWMutex{},
		RegisterServices: make(map[string]*grpc.ClientConn),
		// Services path <host>:<port>/<service_name> => <path>
		// (127.0.0.1:54322/A => /A.A/CallA)
		lockForPather:  &sync.RWMutex{},
		PathToServices: make(map[string][]string),
		// Invocker link
		Invoke:             grpc.Invoke,
		// DB link
		DB:                 DB,
		Dial:               grpc.Dial,
		// default timeouts
		defaultTimeout:     defaultTimeout,
		defaultLoopTimeout: defaultLoopTimeout,
	}

	for _, opt := range opts {
		opt(srv)
	}

	return srv

}

// zexServer structure
type zexServer struct {
	lockForRegger    *sync.RWMutex
	RegisterServices map[string]*grpc.ClientConn

	lockForPather  *sync.RWMutex
	PathToServices map[string][]string

	Invoke Invoker
	Dial   Dialer
	DB     storage.Database

	defaultTimeout     time.Duration
	defaultLoopTimeout time.Duration
}

// Register services interface impl
func (s *zexServer) Register(ctx context.Context, service *zex.Service) (*zex.Empty, error) {
	grpclog.Printf("Start registraion services in Zex (%s, %s)", service.Name, service.Addr)
	// ---------------------
	// Do reflection request
	// ---------------------
	serviceKey := service.Addr + "/" + service.Name
	grpclog.Printf("start get all methods in registred \"%s\" services", serviceKey)

	// get host, port
	ss := strings.Split(service.Addr, ":")

	var host string
	if len(ss) == 2 {
		host, _ = ss[0], ss[1]
	} else {
		return nil, errors.New("incorrect format host:port")
	}

	// create connect
	conn, err := s.Dial(service.Addr, grpc.WithInsecure())
	if err != nil {
		grpclog.Printf("did not connect: %v", err)
		conn.Close()
		return nil, err
	}

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

	// Load FileDescriptorProto from services
	for _, srv := range info.GetListServicesResponse().GetService() {
		if srv.Name != "grpc.reflection.v1alpha.ServerReflection" && srv.Name != "zex.Zex" {
			err = informer.Send(&rpb.ServerReflectionRequest{
				Host:           host,
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

	grpclog.Printf("Add services to map with key %s successed", serviceKey)
	return &zex.Empty{}, nil
}

// Pipeline services interface impl
func (s *zexServer) Pipeline(stream zex.Zex_PipelineServer) error {
	grpclog.Printf("listen stream pipeline")

	pid := uuid.New()
	statusPidKey := []byte(statusPrefix + pid)

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

			// set transaction atomic
			err = transation.Commit()

			if err != nil {
				grpclog.Printf("can't set value to leveldb %s: %v", pipeline, err)
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
			transation.Put([]byte(pid + "_" + cmd.Path), cmd.Body)
			transation.Put(statusPidKey, make([]byte, 0))
			pipeline = append(pipeline, cmd)
		}
	}

	return nil
}

// Subscribe services interface impl
func (s *zexServer) Subscribe(ctx context.Context, pid *zex.Pid) (*zex.Empty, error) {
	grpclog.Printf("subscribe uuid %s", pid.ID)

	pidStatusKey := []byte(statusPrefix + pid.ID)
	timeout := time.After(s.defaultTimeout)

	if s.isExistsPid(pid.ID) {
		// If pid exists, need check status error or invoke
		for {
			// Add sleep for polling
			value, err := s.DB.Get(pidStatusKey, nil)
			if s.DB.IsErrNotFound(err) {
				grpclog.Println("subscribe return answer with status nil, all pipeline done correct")
				return &zex.Empty{}, nil
			}

			if len(string(value)) != 0 {
				grpclog.Printf("subscribe return answer with error: %s", string(string(value)))
				return &zex.Empty{}, errors.New(string(value))
			}
			// check timeout and context cancel
			select {
			case <-ctx.Done():
				grpclog.Printf("contect canceled, return error")
				return &zex.Empty{}, errors.New("context cancel")
			case <-timeout:
				grpclog.Printf("timeout, more when %s return error", s.defaultTimeout)
				return &zex.Empty{}, errors.New("timeout")
			default:
				// Add sleep for polling
				time.Sleep(s.defaultLoopTimeout)
			}
		}
	}
	// If pid does not exist, this is done correct
	return &zex.Empty{}, nil
}

func (s *zexServer) isExistsPid(pid string) bool {
	iter := s.DB.GetIterator(pid, "")
	for iter.Next() {
		key := iter.Key()
		if strings.Contains(string(key), pid) {
			return true
		}
	}
	iter.Release()
	return false
}

// Run pipeline
func (s *zexServer) runPipeline(pid string) {
	grpclog.Printf("Start RunPipeline uuid %s, leveldb count %d", pid, s.DB.GetRowsCount())

	// Get context for cancel all goroutine calls
	var (
		ctx, cancel = context.WithCancel(context.Background())
		pipeline    []zex.Cmd
		statusPidKey = []byte(statusPrefix + pid)
	)

	// create batch transaction
	transationDel := s.DB.NewTransaction()

	// iterate to leveldb
	iter := s.DB.GetIterator(pid, "")
	transationDel.Delete(statusPidKey)
	for iter.Next() {
		key := iter.Key()
		strKey := string(key)
		// collect all pipelines
		value := iter.Value()

		// bad
		// body := value

		// good
		body := make([]byte, len(value))
		copy(body, value)

		cmd := zex.Cmd{
			zex.CmdType(1), strings.Split(strKey, "_")[1],
			body}

		pipeline = append(pipeline, cmd)

		transationDel.Delete(key)
	}
	iter.Release()

	// get count pipelines, for check all goroutines
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
	// wait done or cancel chan

	for err := range errC {
		lengthPipiline--

		// incorrect requests
		if err != nil {
			grpclog.Printf("command was failed by id %s: %s", pid, err)
			// Update status if incorrect request
			s.DB.Put(statusPidKey, []byte(err.Error()), nil)
			// If one of this fail, all failed
			cancel()
			return
		}

		// correct requests
		if lengthPipiline == 0 {
			grpclog.Printf("all command done")
			close(errC)
			err := transationDel.Commit()
			if err != nil {
				grpclog.Printf("level db pid %s has incorrect status %s", pid, err)
				return
			}
		}
	}

	grpclog.Printf("runPipeline %s was done, leveldb cleanup, all rows %d", pid, s.DB.GetRowsCount())

}

// Call command services
func (s *zexServer) callCmd(ctx context.Context, cmd zex.Cmd, errC chan error) {
	var (
		out        = &any.Any{}
		cc         *grpc.ClientConn
		serviceKey []string
		ok         bool
	)

	s.lockForPather.RLock()
	serviceKey, err := s.PathToServices[cmd.Path]
	s.lockForPather.RUnlock()

	if !err {
		errC <- errors.New("incorrect get serviceKey")
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

	in := byteProto(cmd.Body)
	// The grpc.Invoke or Mock in test returns an error, or nil to chan
	ierror := s.Invoke(ctx, cmd.Path, in, out, cc)
	if ierror != nil {
		grpclog.Printf("incorrect invoke %s", ierror)
	}
	errC <- ierror
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
