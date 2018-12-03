package main

import (
	"flag"

	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/local-controller-plugin/controller"
	. "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grpc_server"
	"github.com/tedsuo/ifrit/sigmon"
	"google.golang.org/grpc"
)

var atAddress = flag.String(
	"listenAddr",
	"0.0.0.0:9860",
	"host:port to serve on",
)

////CreateVolume will have been defined under controller.

func main() {
	parseCommandLine()

	logger, _ := lagerflags.NewFromConfig("local-contoller-plugin", lagerflags.ConfigFromFlags())
	logger.Info("starting")
	defer logger.Info("end")

	listenAddress := *atAddress

	controller := controller.NewController(&osshim.OsShim{}, &filepathshim.FilepathShim{}, "")
	server := grpc_server.NewGRPCServer(listenAddress, nil, controller, RegisterServices)

	monitor := ifrit.Invoke(sigmon.New(server))
	logger.Info("started")

	err := <-monitor.Wait()

	if err != nil {
		logger.Fatal("exited-with-failure", err)
	}
}

func parseCommandLine() {
	lagerflags.AddFlags(flag.CommandLine)
	flag.Parse()
}

func RegisterServices(s *grpc.Server, srv interface{}) {
	RegisterControllerServer(s, srv.(ControllerServer))
	RegisterIdentityServer(s, srv.(IdentityServer))
}
