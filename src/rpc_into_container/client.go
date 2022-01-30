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


//server.go

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
