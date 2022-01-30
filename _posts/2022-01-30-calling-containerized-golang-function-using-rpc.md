---
layout: post
title:  "Calling a containerized Golang function using RPC"
date:   2022-01-30 14:26:45 +0300 
tags: docker golang rpc
---

This article demonstrates how to call an exported function of a go program running inside a [docker](https://docs.docker.com/) container from the host PC using [RPC](https://en.wikipedia.org/wiki/Remote_procedure_call).

## The Demo

The demo code involves:

- RPC server that receives & executes shell commands inside a docker container & writes results back to a client

```golang
//server.go (not valid code)

//CmdExecutor cmd process response
type CmdExecutorRes struct{
	Stdout string 
	Stderr string 
	Err error
}

//CmdExecutor exported RPC object for executing client commands
type CmdExecutor interface{
	//Run executes client cmd in blocking mode 
	//then returns cmd process stdout/stderr & ret code
	Run(cmd []string, res *CmdExecutorRes) error
} 

func RunServer() {
	//test code
	cmdE := new(CmdExecutorImpl)

	//register CmdExecutor objecy
	rpc.Register(cmdE)
	rpc.HandleHTTP()
	rpc.Serve(":3000")
}
```

- A client that spawns the containerized RPC server & calls its Run function to execute shell commands in the docker container.

```golang
//client.go

//CmdExecutorContainer for starting CmdExecutor RPC server 
// in a docker container
type CmdExecutorContainer interface{
    //Start CmdExecutor rpc server container
    Start()(err error)
    //Stop CmdExecutor container 
    Stop()(err error)
    //GetIP of RPC server container
    GetIP()(ip string, err error);
}

func RunClient() {
    cmdExecutorContainer :=  &CmdExecutorContainerImpl{}
    cmd := []string{"echo", "Hello From Container"}
    cmdERes := CmdExecutorRes{}

    //spawn container
    cmdExecutorContainer.Start()
    //get ip of container
    containerIP, err := cmdExecutorContainer.GetIP(); 

    //make RPC call
    rpc.Conn("tcp", containerIP+":3000")
    rpc.Call("CmdExecutorImpl.Run", cmd, &cmdERes)
    //log response
    log.Printf(cmdERes.Stdout,cmdERes.Stderr,cmdERes.Err)
    cmdExecutorContainer.Stop()
}
```

Even though this demo is in golang it should be possible to reproduce this in any language that has RPC support (for example [gRPC](https://grpc.io/)) & [docker engine sdk](https://docs.docker.com/engine/api/sdk/). You could write a wrapper around docker binaries if no SDK is available.

This article does not explain how RPC or docker works, you can see some links at the end of this article for those.

## Project Structure / Files

- `server.go` will contain our CmdExecutor RPC Server

```golang
package main

func RunServer() {
}
```

- `client.go` will spawn CmdExecutor in a container & call its Run function via RPC to execute commands inside the container

```golang
package main

func RunClient() {
}
```
- `main.go`

```golang
package main

import (
    "log"
    "os"
)

func main() {
    if len(os.Args) != 2 || (os.Args[1] != "server" && os.Args[1] != "client") {
        log.Fatal("Usage: ./main <server | client>")
    }

    if os.Args[1] == "server" {
        RunServer()
    } else {
        RunClient()
    }
}
```

- `Dockerfile` An ubuntu based image for our CMDExecutor RPC server. Make sure your compiled binary can run in your base image platform.

```Dockerfile
FROM ubuntu:20.04
COPY cmdExecutor ./
EXPOSE 3000
CMD [ "./cmdExecutor", "server"]
```

```bash
#init project
go mod init
#build go binary
go build -o cmdExecutor
#build docker image
docker build -t "cmdexecutor" .
```

## Executing Shell Commands from Golang

We will first need to write a simple command executor that will live inside our docker container to execute client’s commands and return response, I will use golang’s [OS](https://pkg.go.dev/os) package to execute the command in blocking mode then write the response back to the client’s struct.

```golang
package main

import (
    "bytes"
    "os/exec"
    "log"
)

//CmdExecutor exported RPC object for executing client commands
type CmdExecutor int 

//CmdExecutor cmd process response
type CmdExecutorRes struct{
    Stdout string 
    Stderr string 
    Err error
}

//Run executes client cmd in blocking mode then returns cmd process stdout/stderr & ret code
func (r *CmdExecutor) Run(cmdStr []string, res *CmdExecutorRes) error{
    log.Print("Received RPC: ", cmdStr)

    cmd :=  exec.Command(cmdStr[0], cmdStr[1:]...)

    stdout :=  bytes.Buffer{}
    stderr := bytes.Buffer{}

    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    res.Err = cmd.Run() //blocking
    if res.Err != nil{
        return res.Err
    }

    res.Stdout = stdout.String()
    res.Stderr = stderr.String()

    return res.Err;
}

func RunServer() {
    //test code
    cmdE := new(CmdExecutor)
    cmdERes := CmdExecutorRes{}
    cmd := []string{"echo", "Hello From Container"}
    cmdE.Run(cmd, &cmdERes)
    log.Printf("stdout: %v\nstderr: %v\nerr: %v\n", cmdERes.Stdout,cmdERes.Stderr,cmdERes.Err)
}
```

```bash
#build
go build -o cmdExecutor
#rebuild docker image
docker build -t "cmdexecutor" .
#run cmdExecutor
docker run cmdexecutor
```

## Add RPC Server

Golang has an inbuilt net [RPC](https://pkg.go.dev/net/rpc) package that enables us to call exported functions of an object remotely.
In our case the object would be CmdExecutor that would enable a client to execute commands inside the container.

```golang

package main

import (
    ...
    "net"
    "net/http"
    "net/rpc"
)

type CmdExecutor int 
type CmdExecutorRes struct{
    ...
}
func (r *CmdExecutor) Run(cmdStr []string, res *CmdExecutorRes) error{
    ...
    return res.Err;
}

func runRPCServer(cmdE *CmdExecutor) error{
    rpc.Register(cmdE)
    rpc.HandleHTTP()

    listener, err := net.Listen("tcp", ":3000")
    if err != nil{
        return err;
    }

    log.Println("RPC Server Running on :3000")
    err = http.Serve(listener, nil)
    return err;
}

func RunServer() {
    //test code
    cmdE := new(CmdExecutor)
    if err := runRPCServer(cmdE); err != nil{
        log.Fatal("RPC Server Error: ", err);
    }
}
```

```bash
# rebuild & run RPC server
go build -o cmdExecutor
docker build -t "cmdexecutor" .
docker run cmdexecutor
```

## Calling Run function via RPC
Now that we have our RPC server running inside the container its time we wrote the client that connects to it and calls the Run function to execute commands inside the docker container.

```golang
package main

import (
    "log"
    "net/rpc"
)

func RunClient() {

    client, err := rpc.DialHTTP("tcp", "127.0.0.1:3000")
    if err != nil{
        log.Fatal("rpc.DialHTTP: ", err)
    }

    cmd := []string{"echo", "Hello From Container"}
    cmdERes := CmdExecutorRes{}
    err = client.Call("CmdExecutor.Run", &cmd, &cmdERes)
    if err != nil{
        log.Fatal("client.Call: ", err)
    }

    log.Printf("stdout: %v\nstderr: %v\nerr: %v\n", cmdERes.Stdout,cmdERes.Stderr,cmdERes.Err)

}
```

You will have to run the server on the host PC for now, till we have a way of getting its docker IP (hint: `docker inspect <container_id>`)

```bash
#build
go build -o cmdExecutor
#run server on host pc 
./cmdExecutor server
# on another terminal
./cmdExecutor client
```

## Spawning RPC container from client & getting its IP

Since our cmd executor will live in a docker container, we need a way to spawn the container & get its IP so we can be able to connect to the RPC server. For this I used [docker golang’s SDK](https://pkg.go.dev/github.com/docker/docker/client)

```golang
package main

import (
    "log"
    "net/rpc"
    "time"
    "errors"
    "context"
    "github.com/docker/docker/api/types"
    "github.com/docker/docker/api/types/container"
    "github.com/docker/docker/client"
)

type CmdExecutorContainer struct{
    ctx context.Context 
    client *client.Client
    ct container.ContainerCreateCreatedBody
}

func (a *CmdExecutorContainer) Start()(err error){

    a.ctx = context.Background()
    a.client, err = client.NewClientWithOpts(client.FromEnv)
    if err != nil{
        return
    }

    a.ct, err = a.client.ContainerCreate(
        a.ctx,
        &container.Config{
        Image: "cmdexecutor:latest",
        },
        nil, nil, nil, "",
    )
    if err != nil{
        return
    }

    err = a.client.ContainerStart(a.ctx, a.ct.ID, types.ContainerStartOptions{})
    return
}

func (a *CmdExecutorContainer) Stop()(err error){
    err = a.client.ContainerStop(a.ctx, a.ct.ID, nil)
    return 
}

func (a *CmdExecutorContainer) GetIP()(ip string, err error){

    res, err := a.client.ContainerInspect(a.ctx, a.ct.ID)
    if err != nil{
        return
    }

    if res.NetworkSettings == nil{
        return ip, errors.New("NetworkSettings nil")
    }

    ip = res.NetworkSettings.IPAddress
    return 
}

func RunClient() {

    cmdExecutorContainer :=  CmdExecutorContainer{}
    cmd := []string{"echo", "Hello From Container"}
    cmdERes := CmdExecutorRes{}

    if err := cmdExecutorContainer.Start(); err != nil{
        log.Fatal("ContainerStart ", err)
    }

    defer cmdExecutorContainer.Stop()
    time.Sleep(time.Second * 1) // let container start, @todo poll docker inspect

    containerIP, err := cmdExecutorContainer.GetIP(); 
    if err != nil{
        log.Println("Err GetIP ", err)
        return
    }
    log.Println("CmdExecutorContainer IP ", containerIP)

    client, err := rpc.DialHTTP("tcp", containerIP+":3000")
    if err != nil{
        log.Println("rpc.DialHTTP: ", err)
        return
    }

    if err = client.Call("CmdExecutor.Run", cmd, &cmdERes); err != nil{
        log.Println("Err client.Call: ", err)
        return
    }
    log.Printf("stdout: %v\nstderr: %v\nerr: %v\n", cmdERes.Stdout,cmdERes.Stderr,cmdERes.Err)

}
```

```bash
#install docker SDK
go mod tidy
#build 
go build -o cmdExecutor
#rebuild image
docker build -t "cmdexecutor" .
#spawn container & call function via RPC demo
./cmdExecutor client
```

I get this results on my PC

```
$ ./cmdExecutor client
CmdExecutorContainer IP  172.17.0.2
stdout: Hello From Container
stderr: 
err: <nil>
```

An interesting exercise to attempt would be to execute the command in non-blocking mode then stream the results (stdout/stderr) back to client as the command executes. Or calling the function from another language (see rpcjson or grpc).

## Resources & Further reading

> Its important to note that a docker process is not a sandbox and should not be used to execute untrusted user code, see projects like gVisor or firecracker to see how to execute untrusted code in a container.

- [https://en.wikipedia.org/wiki/Remote_procedure_call](https://en.wikipedia.org/wiki/Remote_procedure_call)
- [https://docs.docker.com/](https://docs.docker.com/)
- [https://pkg.go.dev/net/rpc](https://pkg.go.dev/net/rpc)

You can find the sources here [https://github.com/jakhax/myblog/tree/master/src/rpc_into_container](https://github.com/jakhax/myblog/tree/master/src/rpc_into_container)