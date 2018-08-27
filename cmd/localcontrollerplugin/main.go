package main

import (
	"flag"
	"log"
	"os"

	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/local-controller-plugin/controller"
	. "github.com/container-storage-interface/spec/lib/go/csi/v0"
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

	logger := lager.NewLogger("local-contoller-plugin")
	sink := lager.NewReconfigurableSink(lager.NewWriterSink(os.Stdout, lager.DEBUG), lager.DEBUG)
	logger.RegisterSink(sink)

	listenAddress := *atAddress

	controller := controller.NewController(&osshim.OsShim{}, &filepathshim.FilepathShim{}, "")
	server := grpc_server.NewGRPCServer(listenAddress, nil, controller, RegisterServices)

	monitor := ifrit.Invoke(sigmon.New(server))
	log.Println("Started")

	err := <-monitor.Wait()

	if err != nil {
		log.Fatalf("exited-with-failure: %v", err)
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
