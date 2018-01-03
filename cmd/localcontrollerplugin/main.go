package main

import (
	"flag"
	"log"
	"os"

	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"

	"github.com/jeffpak/local-controller-plugin/controller"
	. "github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grpc_server"
	"github.com/tedsuo/ifrit/sigmon"
	"code.cloudfoundry.org/lager/lagerflags"
)

var atAddress = flag.String(
	"listenAddr",
	"0.0.0.0:50051",
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
	server := grpc_server.NewGRPCServer(listenAddress, nil, controller, RegisterControllerServer)

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
