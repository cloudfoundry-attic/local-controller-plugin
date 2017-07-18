package main

import (
	"log"
	"net"

	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/osshim"

	csi "github.com/jeffpak/csi"
	"github.com/jeffpak/local-controller-plugin/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":50051"
)

//
////CreateVolume will have been defined under models.

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	controller := models.NewController(&osshim.OsShim{}, &filepathshim.FilepathShim{}, "")
	csi.RegisterControllerServer(s, controller)

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
