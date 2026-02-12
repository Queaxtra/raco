package protocol

import (
	"raco/protocol/func/grpc"
)

func NewGRPCClient(address string) StreamHandler {
	return grpc.NewClient(address)
}
