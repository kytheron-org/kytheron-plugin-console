package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kytheron-org/kytheron-plugin-framework/listener"
	pb "github.com/kytheron-org/kytheron-plugin-go/plugin"

	"google.golang.org/grpc"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
)

type console struct {
	pb.UnimplementedOutputPluginServer
	pb.UnimplementedPluginServer
}

func (c *console) GetMetadata(ctx context.Context, empty *pb.Empty) (*pb.Metadata, error) {
	return &pb.Metadata{}, nil
}

func (c *console) Configure(ctx context.Context, req *pb.ConfigureRequest) (*pb.ConfigureResponse, error) {
	return &pb.ConfigureResponse{}, nil
}

func (c *console) Trigger(ctx context.Context, req *pb.EvaluationRequest) (*pb.EvaluationResponse, error) {
	output := map[string]interface{}{
		"logs":        req.Logs,
		"policy_name": req.PolicyName,
	}
	contents, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(contents))
	return &pb.EvaluationResponse{}, nil
}

func main() {
	plugin := &console{}

	l, err := listener.NewSocket(os.Getenv("PLUGIN_UNIX_SOCKET_DIR"), "console")
	if err != nil {
		panic(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	var maxMessageSize int = math.MaxInt
	grpcServer := grpc.NewServer(
		grpc.MaxSendMsgSize(maxMessageSize), // 50MB example
		grpc.MaxRecvMsgSize(maxMessageSize),
	)
	pb.RegisterOutputPluginServer(grpcServer, plugin)
	pb.RegisterPluginServer(grpcServer, plugin)

	go func() {
		log.Printf("gRPC server listening on %v", l.Addr())
		if err := grpcServer.Serve(l); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	go func() {
		select {
		case sig := <-sigs:
			println("Received signal:", sig)
			done <- true // Signal main goroutine to exit
		}
	}()

	handshake := map[string]interface{}{
		"type": "handshake",
		"addr": l.Addr().String(),
	}
	contents, err := json.Marshal(handshake)
	fmt.Println(string(contents))
	<-done
}
