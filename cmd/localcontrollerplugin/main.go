package main

import (
  "net"
  "log"
  "google.golang.org/grpc/reflection"
)

import (
  "google.golang.org/grpc"

  //TODO: CHANGE TO CONTROLLER SERVER
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

  // Register reflection service on gRPC server.
  reflection.Register(s)
  if err := s.Serve(lis); err != nil {
    log.Fatalf("failed to serve: %v", err)
  }
}