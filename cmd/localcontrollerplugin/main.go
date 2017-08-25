package main

import (
	"flag"
	"log"
	"os"

	cf_lager "code.cloudfoundry.org/cflager"
	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"

	"github.com/jeffpak/local-controller-plugin/controller"
	. "github.com/paulcwarren/spec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grpc_server"
	"github.com/tedsuo/ifrit/sigmon"
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
	cf_lager.AddFlags(flag.CommandLine)
	flag.Parse()
}
