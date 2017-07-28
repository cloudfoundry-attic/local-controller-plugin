package main

import (
	"fmt"
	"log"

	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/osshim"

	. "github.com/container-storage-interface/spec"
	"github.com/jeffpak/local-controller-plugin/controller"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grpc_server"
)

const (
	port = 50051
)

////CreateVolume will have been defined under controller.

func main() {
	listenAddress := fmt.Sprintf("0.0.0.0:%d", port)

	controller := controller.NewController(&osshim.OsShim{}, &filepathshim.FilepathShim{}, "")
	server := grpc_server.NewGRPCServer(listenAddress, nil, controller, RegisterControllerServer)

	monitor := ifrit.Invoke(server)
	log.Println("Started")

	err := <-monitor.Wait()

	if err != nil {
		log.Fatalf("exited-with-failure: %v", err)
	}
}
