# zex
Task proxymanager for grpc services


 ~/work/go/src/github.com/zex/zex $ protoc -I . proto/zex.proto --go_out=plugins=grpc:.
 
 Install protobuf 
 
# Make sure you grab the latest version
curl -OL https://github.com/google/protobuf/releases/download/v3.2.0/protoc-3.2.0-linux-x86_64.zip

# Unzip
unzip protoc-3.2.0-linux-x86_64.zip -d protoc3

# Move only protoc* to /usr/bin/
sudo mv protoc3/bin/protoc /usr/bin/protoc
