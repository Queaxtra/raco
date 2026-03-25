package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"os"
	"path/filepath"
	"raco/util"
	"regexp"
	"strings"
	"time"
)

func runGRPCReflect(args []string) int {
	fs := flag.NewFlagSet("grpc-reflect", flag.ContinueOnError)
	address := fs.String("r", "", "gRPC server address (host:port)")
	insecureMode := fs.Bool("insecure", false, "Use insecure connection")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if *address == "" {
		fmt.Fprintln(os.Stderr, "Usage: raco grpc reflect -r <address> [-insecure]")
		return 1
	}
	if !util.ValidateGRPCTarget(*address) {
		fmt.Fprintln(os.Stderr, "Error: invalid gRPC target")
		return 1
	}
	if *insecureMode {
		fmt.Fprintln(os.Stderr, "Error: insecure gRPC reflection is disabled")
		return 1
	}

	// Reflection talks to arbitrary remote services, so target validation and TLS
	// enforcement happen before the connection attempt.
	creds := credentials.NewTLS(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, *address, grpc.WithTransportCredentials(creds), grpc.WithBlock())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	defer conn.Close()

	client := reflectionpb.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if err := stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{},
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	resp, err := stream.Recv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	services := resp.GetListServicesResponse().Service
	for _, service := range services {
		fmt.Println(service.Name)
	}
	return 0
}

// runGRPCScaffold generates a minimal request envelope from a local proto file.
// The output path is kept inside the workspace to avoid arbitrary file writes.
func runGRPCScaffold(args []string) int {
	fs := flag.NewFlagSet("grpc-scaffold", flag.ContinueOnError)
	protoPath := fs.String("proto", "", "Proto file path")
	service := fs.String("service", "", "Service name")
	method := fs.String("method", "", "Method name")
	outputPath := fs.String("o", "", "Output file path")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if *protoPath == "" || *service == "" || *method == "" {
		fmt.Fprintln(os.Stderr, "Usage: raco grpc scaffold --proto <file> --service <svc> --method <method> [-o <file>]")
		return 1
	}

	envelope, err := scaffoldFromProto(*protoPath, *service, *method)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	data, _ := json.MarshalIndent(envelope, "", "  ")
	if *outputPath == "" {
		fmt.Println(string(data))
		return 0
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	cleanPath, err := util.ResolveContainedPath(cwd, *outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0750); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if err := os.WriteFile(cleanPath, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("Wrote scaffold to %s\n", cleanPath)
	return 0
}

// scaffoldFromProto is intentionally lightweight: it extracts enough structure
// for operator productivity without introducing a full protobuf compiler path.
func scaffoldFromProto(protoPath string, serviceName string, methodName string) (map[string]interface{}, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cleanPath, err := util.ResolveContainedPath(cwd, protoPath)
	if err != nil {
		return nil, err
	}
	data, err := util.ReadFileBounded(cleanPath, 1024*1024)
	if err != nil {
		return nil, err
	}
	content := string(data)
	rpcPattern := regexp.MustCompile(`rpc\s+` + regexp.QuoteMeta(methodName) + `\s*\(\s*([A-Za-z0-9_\.]+)\s*\)`)
	rpcMatch := rpcPattern.FindStringSubmatch(content)
	if len(rpcMatch) < 2 {
		return nil, fmt.Errorf("method not found in proto: %s", methodName)
	}
	messageName := rpcMatch[1]
	messagePattern := regexp.MustCompile(`message\s+` + regexp.QuoteMeta(simpleProtoName(messageName)) + `\s*\{([^}]*)\}`)
	messageMatch := messagePattern.FindStringSubmatch(content)
	payload := make(map[string]interface{})
	if len(messageMatch) >= 2 {
		fieldPattern := regexp.MustCompile(`\s*(?:repeated\s+)?[A-Za-z0-9_\.]+\s+([A-Za-z0-9_]+)\s*=\s*\d+`)
		fieldMatches := fieldPattern.FindAllStringSubmatch(messageMatch[1], -1)
		for _, field := range fieldMatches {
			if len(field) < 2 {
				continue
			}
			payload[field[1]] = ""
		}
	}
	return map[string]interface{}{
		"service":  serviceName,
		"method":   methodName,
		"payload":  payload,
		"metadata": map[string]string{},
	}, nil
}

func simpleProtoName(name string) string {
	lastDot := strings.LastIndex(name, ".")
	if lastDot == -1 {
		return name
	}
	return name[lastDot+1:]
}
