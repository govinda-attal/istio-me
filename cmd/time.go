package cmd

import (
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/spf13/cobra"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/govinda-attal/istio-me/internal/handler"
	"github.com/govinda-attal/istio-me/pkg/trials"
)

// timeCmd represents the time command
var timeCmd = &cobra.Command{
	Use:   "time",
	Short: "Launches the time grpc server and rpc server on 8080",
	Run: func(cmd *cobra.Command, args []string) {
		runTimeSrv()
	},
}

func init() {
	rootCmd.AddCommand(timeCmd)
}

func runTimeSrv() {

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		panic(err)
	}

	tracer, closer, err := cfg.New(
		"timer",
		config.Logger(jaeger.StdLogger),
	)
	if err != nil {
		panic(err)
	}
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	// Timer Service Client Configuration
	//var conn *grpc.ClientConn
	conn, err := grpc.Dial(":8080",
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer)))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	var opts []grpc.ServerOption
	opts = append(opts, grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)))
	grpcServer := grpc.NewServer(opts...)
	trials.RegisterTimerServer(grpcServer, handler.NewTimerSrv())

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("microservice started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	grpcServer.GracefulStop()
	log.Println("microservice shutdown")
}
