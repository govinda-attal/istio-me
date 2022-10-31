package cmd

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	// otgrpc "github.com/opentracing-contrib/go-grpc"
	//opentracing "github.com/opentracing/opentracing-go"
	ot "github.com/opentracing/opentracing-go"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/spf13/cobra"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	// "github.com/uber/jaeger-client-go/zipkin"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"

	"github.com/govinda-attal/istio-me/internal/handler"
	"github.com/govinda-attal/istio-me/pkg/trials"
)

func UnaryServerMetdataTagInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if ctxMd, ok := metadata.FromIncomingContext(ctx); ok {
			for key, field := range ctxMd {
				log.Println(key, field)
			}
		}
		return handler(ctx, req)
	}
}


// greetCmd represents the greet command
var greetCmd = &cobra.Command{
	Use:   "greet",
	Short: "Launches the grpc greet server and rpc server on 9080",
	Run: func(cmd *cobra.Command, args []string) {
		runGreetSrv()
	},
}

func init() {
	rootCmd.AddCommand(greetCmd)
}

func runGreetSrv() {

	// zipkinPropagator := zipkin.NewZipkinB3HTTPHeaderPropagator()
	// injector := jaeger.TracerOptions.Injector(opentracing.HTTPHeaders, zipkinPropagator)
	// extractor := jaeger.TracerOptions.Extractor(opentracing.HTTPHeaders, zipkinPropagator)

	// // Zipkin shares span ID between client and server spans; it must be enabled via the following option.
	// zipkinSharedRPCSpan := jaeger.TracerOptions.ZipkinSharedRPCSpan(true)

	// sender, _ := jaeger.NewUDPTransport("jaeger-agent.istio-system:5775", 0)
	// tracer, closer := jaeger.NewTracer(
	// 	"greeter",
	// 	jaeger.NewConstSampler(true),
	// 	jaeger.NewRemoteReporter(
	// 		sender,
	// 		jaeger.ReporterOptions.BufferFlushInterval(1*time.Second)),
	// 	injector,
	// 	extractor,
	// 	zipkinSharedRPCSpan,
	// )

	// opentracing.SetGlobalTracer(tracer)
	// defer closer.Close()

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		panic(err)
	}

	tracer, closer, err := cfg.New(
		"greeter",
		config.Logger(jaeger.StdLogger),
	)
	if err != nil {
		panic(err)
	}
	ot.SetGlobalTracer(tracer)
	defer closer.Close()

	// Timer Service Client Configuration
	//var conn *grpc.ClientConn
	conn, err := grpc.Dial("timer:8080",
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_opentracing.UnaryClientInterceptor(grpc_opentracing.WithTracer(tracer))))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	tc := trials.NewTimerClient(conn)

	var opts []grpc.ServerOption

	opts = append(opts, 
		grpc.UnaryInterceptor(grpc_opentracing.UnaryServerInterceptor(grpc_opentracing.WithTracer(tracer))),		
	)
	grpcServer := grpc.NewServer(opts...)
	trials.RegisterGreeterServer(grpcServer, handler.NewGreeterSrv(tc))

	lis, err := net.Listen("tcp", ":9080")
	if err != nil {
		panic(err)
	}

	// Web Server Configuration
	ctx := context.Background()
	dopts := []grpc.DialOption{grpc.WithInsecure()}
	gwmux := runtime.NewServeMux()
	err = trials.RegisterGreeterHandlerFromEndpoint(ctx, gwmux, "localhost:9080", dopts)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", gwmux)
	mux.Handle("/spec/greeter.swagger.json", http.StripPrefix("/spec", http.FileServer(http.Dir("api"))))

	// Special HTTP Server that handles the traffic routes either to GRPC or RESTful Gateway
	srv := &http.Server{
		Addr: ":9080",
		Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
				grpcServer.ServeHTTP(w, r)
			} else {
				mux.ServeHTTP(w, r)
			}
		}), &http2.Server{}),
	}

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("microservice started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)
	grpcServer.GracefulStop()
	log.Println("microservice shutdown")
}
