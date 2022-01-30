package main

import (
	"bytes"
	"os/exec"
	"log"
	"net"
	"net/http"
	"net/rpc"
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
