## Building & Running

```bash
#install dependencies
go mod tidy
#build golang binary
go build -o  cmdExecutor
#build image
docker build -t "cmdexecutor" .
#spawn cmdexecutor container & call Run function via RPC
./cmdExecutor client