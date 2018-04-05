package control

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"

	logService "v2ray.com/core/app/log/command"
	statsService "v2ray.com/core/app/stats/command"
	"v2ray.com/core/common"
)

func getServiceMethod(s string) (string, string) {
	ss := strings.Split(s, ".")
	service := ss[0]
	var method string
	if len(ss) > 1 {
		method = ss[1]
	}
	return service, method
}

func printUsage() {
	fmt.Println("v2ctl api [--server=127.0.0.1:8080] Service.Method Request")
	fmt.Println("Call an API in an V2Ray process.")
	fmt.Println("The following methods are currently supported:")
	fmt.Println("\tLoggerService.RestartLogger")
	fmt.Println("\tStatsService.GetStats")
}

type serviceHandler func(conn *grpc.ClientConn, method string, request string) (string, error)

var serivceHandlerMap = map[string]serviceHandler{
	"statsservice":  callStatsService,
	"loggerservice": callLogService,
}

func callLogService(conn *grpc.ClientConn, method string, request string) (string, error) {
	client := logService.NewLoggerServiceClient(conn)

	switch strings.ToLower(method) {
	case "restartlogger":
		r := &logService.RestartLoggerRequest{}
		if err := proto.UnmarshalText(request, r); err != nil {
			return "", err
		}
		resp, err := client.RestartLogger(context.Background(), r)
		if err != nil {
			return "", err
		}
		return proto.MarshalTextString(resp), nil
	default:
		return "", errors.New("Unknown method: " + method)
	}
}

func callStatsService(conn *grpc.ClientConn, method string, request string) (string, error) {
	client := statsService.NewStatsServiceClient(conn)

	switch strings.ToLower(method) {
	case "getstats":
		r := &statsService.GetStatsRequest{}
		if err := proto.UnmarshalText(request, r); err != nil {
			return "", err
		}
		resp, err := client.GetStats(context.Background(), r)
		if err != nil {
			return "", err
		}
		return proto.MarshalTextString(resp), nil
	default:
		return "", errors.New("Unknown method: " + method)
	}
}

func init() {
	const name = "api"
	common.Must(RegisterCommand(name, "Call V2Ray API", func(arg []string) {
		fs := flag.NewFlagSet(name, flag.ContinueOnError)

		serverAddrPtr := fs.String("server", "127.0.0.1:8080", "Server address")

		err := fs.Parse(arg)
		switch err {
		case nil:
		case flag.ErrHelp:
			printUsage()
			return
		default:
			fmt.Fprintln(os.Stderr, "Error parsing arguments:", err)
			return
		}

		conn, err := grpc.Dial(*serverAddrPtr, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to dial", *serverAddrPtr, ":", err)
		}
		defer conn.Close()

		unnamedArgs := fs.Args()
		if len(unnamedArgs) < 2 {
			printUsage()
			return
		}

		service, method := getServiceMethod(unnamedArgs[0])
		handler, found := serivceHandlerMap[strings.ToLower(service)]
		if !found {
			fmt.Fprintln(os.Stderr, "Unknown service:", service)
			return
		}

		response, err := handler(conn, method, unnamedArgs[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to call service", unnamedArgs[0], ":", err)
			return
		}

		fmt.Println(response)
	}))
}