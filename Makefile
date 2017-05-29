run_zex:
	go run cmd/zex/main.go

run_a:
	go run cmd/a/main.go

run_b:
	go run cmd/b/main.go

run_c:
	go run cmd/c/main.go

run_client:
	go run cmd/client/main.go

test:
	go test ./server -v

test_bench:
	go test ./server -bench=. -v

requires:
	github.com/golang/protobuf/{proto,protoc-gen-go}
	go get -u golang.org/x/net/context
	go get -u google.golang.org/grpc
	go get -u google.golang.org/grpc/metadata
	go get -u github.com/garyburd/redigo/redis
	go get -u github.com/pborman/uuid
	go get -u github.com/syndtr/goleveldb/leveldb

bootstrap: requires