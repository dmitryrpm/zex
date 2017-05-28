package server

import (
	"fmt"
	"github.com/dmitryrpm/zex/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"google.golang.org/grpc/grpclog"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/dmitryrpm/zex/storage_mock"
)


type mockInvoker struct {
	lock *sync.Mutex
	data []string

	letency   time.Duration
	callCount int
	errCount  int
	err       error
}

const timeLetency = 1 * time.Microsecond

func (m *mockInvoker) Invoke(ctx context.Context, method string, args, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	var (
		err error
		str = method + "(" + args.(fmt.Stringer).String() + ")"
	)

	m.lock.Lock()
	m.callCount++
	if (m.errCount == 0 || m.callCount == m.errCount) && m.err != nil {
		err = m.err
	}
	m.lock.Unlock()

	// if for test need emulations of net latency
	if m.letency > 0 {
		grpclog.Printf("[test] set letency %s", m.letency)
		time.Sleep(m.letency)
		grpclog.Printf("wait %s", m.letency)
	}


	// other routes cancellation
	if err != nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}

	str = fmt.Sprintf("%s->%s", str, err)
	m.lock.Lock()
	m.data = append(m.data, str)
	m.lock.Unlock()

	return err
}


type MockZexServer struct {
	desc string

	invokeLetency time.Duration // help for cancel when err cancel
	invokeErr     error
	invokeErrWhen int
	pid           string

	setPathToServices   map[string][]string
	setRegisterServices map[string]*grpc.ClientConn
	pipeline            []*zex.Cmd

	countRows int
	expCallerCmd []string
}

type MockDbLevel struct {
	batch *leveldb.Batch
}


func (db *MockDbLevel) Write(batch *leveldb.Batch, wo *opt.WriteOptions) error {
	if batch == nil || batch.Len() == 0 {
		return errors.New("batch null")
	}
	return nil
}

func TestUnits(t *testing.T) {

	errStrLetency := fmt.Sprintf("waiting %s", timeLetency)

	zexMocks := []MockZexServer{
		{
			desc: "test Pipeline + runPipeline success simple",
			pid: "pid-1",
			setPathToServices: map[string][]string{
				"/A.A/CallC": []string{"localhost:2345"},
				"/A.A/CallB": []string{"localhost:2345"},
			},
			setRegisterServices: map[string]*grpc.ClientConn{
				"localhost:2345": &grpc.ClientConn{},
			},
			pipeline: []*zex.Cmd{
				{
					Path: "/A.A/CallC",
					Body: []byte("aaaa"),
				},
				{
					Path: "/A.A/CallB",
					Body: []byte("aaaa"),
				},
			},
			expCallerCmd: []string{
				"/A.A/CallC(aaaa)->%!s(<nil>)",
				"/A.A/CallB(aaaa)->%!s(<nil>)",
			},
			countRows: 0,

		},
		{
			desc: fmt.Sprintf("test Pipeline + runPipeoine fail with letency %s", errStrLetency),
			pid: "pid-2",
			invokeLetency: timeLetency,
			invokeErr: errors.New(errStrLetency),
			setPathToServices: map[string][]string{
				"/A.A/CallC": []string{"localhost:2345"},
			},
			setRegisterServices: map[string]*grpc.ClientConn{
				"localhost:2345": &grpc.ClientConn{},
			},
			pipeline: []*zex.Cmd{
				{
					Path: "/A.A/CallC",
					Body: []byte("aaaa"),
				},
			},
			countRows: 1,
			expCallerCmd: []string{
				fmt.Sprintf("/A.A/CallC(aaaa)->%s", errStrLetency),
			},
		},
	}

	for _, tc := range zexMocks {
		t.Run(tc.desc, func(tt *testing.T) {
			m := &mockInvoker{
				lock:     &sync.Mutex{},
				errCount: tc.invokeErrWhen,
				err:      tc.invokeErr,
				letency:  tc.invokeLetency,
			}


			// example for show how work with options
			storageMock, _ := storage_mock.NewMock("test")
			impl := NewMock(m.Invoke, storageMock)
			impl.PathToServices = tc.setPathToServices
			impl.RegisterServices = tc.setRegisterServices

			tr := storageMock.NewTransaction()
			for _, cmd := range tc.pipeline {
				str := tc.pid + "_" + cmd.Path
				tr.Put([]byte(str), []byte(cmd.Body))
			}
			tr.Commit()

			impl.runPipeline(tc.pid)
			count := impl.DB.GetRowsCount()
			if  count != tc.countRows {
				tt.Errorf("storage_mock rows shoude be %s," +
					" but we have rows \"%v\"", tc.countRows, count)
			}

			// sort expected
			sort.Strings(tc.expCallerCmd)
			// sort called
			sort.Strings(m.data)
			if strings.Join(tc.expCallerCmd, ",") != strings.Join(m.data, ",") {
				tt.Errorf("expected equals, but \"%v\" != \"%v\"", tc.expCallerCmd, m.data)
			}
		})
	}


	t.Run("test subscribe services", func(tt *testing.T) {
		tt.Skip("need add test")
	})

	t.Run("test registry services", func(tt *testing.T) {
		tt.Skip("need add test")
	})
}