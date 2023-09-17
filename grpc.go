package uker

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	grpcp "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Struct with all fields to connect to gRPC server
type GrpcConnData struct {
	Host     string
	Port     string
	KeyPath  string
	CertPath string
}

// Struct with all fields to set up gRPC server
type GrpcServerData struct {
	Port     string
	KeyPath  string
	CertPath string
}

// Global interface
type grpc interface {
	CreateClient(GrpcConnData) *grpcp.ClientConn
	CreateServer(*sync.WaitGroup, GrpcServerData) *grpcp.Server
}

// Local struct to be implmented
type grpc_implementation struct{}

// External contructor
func NewGrpc() grpc {
	return &grpc_implementation{}
}

// Create gRPC server
//
// @param wg *sync.WaitGroup: WaitGroup to add the server routine.
//
// @param sData GrpcServerData: Struct with necessary data to set up the gRPC server.
//
// @return *grpcp.Server: created & running gRPC.Server pointer
func (g *grpc_implementation) CreateServer(wg *sync.WaitGroup, sData GrpcServerData) *grpcp.Server {
	// Create TLS credentials
	creds, err := credentials.NewServerTLSFromFile(sData.CertPath, sData.KeyPath)
	if err != nil {
		panic(fmt.Sprintf("cannot load TLS credentials: %s", err.Error()))
	}

	gRPCServer := grpcp.NewServer(grpcp.Creds(creds))

	// Adding routine to wg
	wg.Add(1)

	// Starting gRPC server
	go func(gRPCServer *grpcp.Server) {
		listener, err := net.Listen("tcp", sData.Port)
		if err != nil {
			panic(fmt.Sprintf("cannot create tcp connection: %s", err.Error()))
		}

		err = gRPCServer.Serve(listener)
		if err != nil {
			panic(fmt.Sprintf("cannot initialize grpc server: %s", err.Error()))
		}

		wg.Done()
	}(gRPCServer)

	return gRPCServer
}

// Create gRPC Client
//
// @param dialData GrpcConnData: WaitGroup to add the server routine.
//
// @return (*grpcp.Server, error): the stablished connection with the server & error if exists
func (g *grpc_implementation) CreateClient(dialData GrpcConnData) *grpcp.ClientConn {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create TLS credentials
	creds, err := credentials.NewServerTLSFromFile(dialData.CertPath, dialData.KeyPath)
	if err != nil {
		panic(fmt.Errorf("cannot load TLS credentials: %v", err))
	}

	// Opening the connection to gRPC server
	conn, err := grpcp.DialContext(ctx, fmt.Sprintf("%s:%s", dialData.Host, dialData.Port), grpcp.WithTransportCredentials(creds), grpcp.WithBlock(), grpcp.WithReturnConnectionError())
	if err != nil {
		panic(fmt.Errorf("cannot connect with gRPC server: %s ", err.Error()))
	}

	return conn
}
