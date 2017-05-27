run:
	go run main.go

requires:
	go get -u github.com/golang/protobuf/proto
	go get -u golang.org/x/net/context
	go get -u google.golang.org/grpc
	go get -u google.golang.org/grpc/metadata
	go get -u github.com/garyburd/redigo/redis
	go get -u github.com/pborman/uuid
	go get -u github.com/syndtr/goleveldb/leveldb

bootstrap: requires