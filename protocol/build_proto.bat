cd ./proto
protoc --proto_path=. --go_out=. *.proto
cd ../excel
protoc --proto_path=. --go_out=. *.proto
cd ../config
protoc --proto_path=. --go_out=. *.proto