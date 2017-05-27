# zex 

Task proxymanager for grpc services

Тестовое задание

Первая реализация сервера ЦЕХ.  Нужно написать сервис, который будет реализовывать следующий GRPC DSL

* через Pipeline можно вызывать только unary методы и нам все равно на их ответ, кроме ошибки (упрощение)
* cценарий выполения при наполенени pipeline cохраняется в goleveldb
* cценарий выполения запускается конкурентно и данные вычитываются из goleveldb
* при получении ошибки все контексты отменяются
* базовый тест мы должны зарегать три сервиса A,B,C и вызвать у каждого по три метода CallA,CallB,CallC в рамках одного пайплайна и должет быть успех
* файл тест мы должны зарегать три сервиса A,B,C и вызвать у каждого по три метода CallA,CallB,CallC и у B CallErr в рамках одного пайплайна и должет быть файл и отмена контекста и вызовов у других
* Body нужно передавать при вызыве сервиса как proto.Marsheler - тут простая обертка над bytes которая возращает сами байты


# Запуск
make requires
make run_zex

### Install grpc
Make sure you grab the latest version
curl -OL https://github.com/google/protobuf/releases/download/v3.2.0/protoc-3.2.0-linux-x86_64.zip
unzip protoc-3.2.0-linux-x86_64.zip -d protoc3
sudo mv protoc3/bin/protoc /usr/bin/protoc

### Install gp-grpc
go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
go get -u google.golang.org/grpc

### Compile grpc-protofiles
protoc --go_out=plugins=grpc:. *.proto
