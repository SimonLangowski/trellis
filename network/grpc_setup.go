package network

// GRPC network setup things

import (
	"crypto/tls"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/simonlangowski/lightning1/config"
	coord "github.com/simonlangowski/lightning1/coordinator/messages"
	"github.com/simonlangowski/lightning1/network/messages"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func RunServer(handler messages.MessageHandlersServer, coordHandler coord.CoordinatorHandlerServer, servercfgs map[int64]*config.Server, addr string) {
	server := StartServer(handler, coordHandler, servercfgs, addr)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	server.Stop()
	log.Printf("Server %v stopped", addr)
}

func StartServer(handler messages.MessageHandlersServer, coordHandler coord.CoordinatorHandlerServer, servercfgs map[int64]*config.Server, addr string) *grpc.Server {
	id, myCfg := FindConfig(addr, servercfgs)
	if id < 0 {
		panic("Could not find " + addr)
	}
	cert, err := tls.X509KeyPair(myCfg.Identity, myCfg.PrivateIdentity)
	if err != nil {
		panic(err)
	}
	cred := credentials.NewServerTLSFromCert(&cert)
	grpcServer := grpc.NewServer(grpc.Creds(cred),
		grpc.MaxRecvMsgSize(2*config.StreamSize), grpc.MaxSendMsgSize(2*config.StreamSize))
	if handler != nil {
		messages.RegisterMessageHandlersServer(grpcServer, handler)
	}
	if coordHandler != nil {
		coord.RegisterCoordinatorHandlerServer(grpcServer, coordHandler)
	}
	lis, err := net.Listen("tcp", config.Port(addr))
	if err != nil {
		log.Fatal("Could not listen:", addr, err)
	}

	go func() {
		err := grpcServer.Serve(lis)
		if err != nil && err != grpc.ErrServerStopped {
			log.Fatal("Serve err:", err)
		}
	}()
	log.Printf("Server %v started", addr)
	return grpcServer
}

func FindConfig(addr string, servercfgs map[int64]*config.Server) (int64, *config.Server) {
	for id, cfg := range servercfgs {
		if cfg.Address == addr {
			return id, cfg
		}
	}
	return -1, nil
}
