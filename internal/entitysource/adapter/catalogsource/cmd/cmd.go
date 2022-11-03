package main

import (
	"github.com/operator-framework/deppy/internal/entitysource/adapter/api"
	"github.com/operator-framework/deppy/internal/entitysource/adapter/catalogsource"

	"github.com/operator-framework/operator-registry/pkg/lib/graceful"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"fmt"
	"net"
	"os"
)

func main() {
	cmd := &cobra.Command{
		Short: "adapter",
		Long:  `runs a deppy adapter that converts CatalogSource contents to deppy source entities`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			return nil
		},
		RunE: runCmdFunc,
	}
	cmd.Flags().StringP("port", "p", "50052", "port number to serve on")
	cmd.Flags().StringP("namespace", "n", "default", "namespace for CatalogSource")
	cmd.Flags().StringP("source", "s", "", "name of CatalogSource")
	cmd.Flags().StringP("address", "a", ":50051", "Address of CatalogSource. Ignored if --namespace and --source and provided")

	if err := cmd.Execute(); err != nil {
		logrus.Errorf("Failed to run deppy source adapter: %v", err)
		os.Exit(1)
	}
}

func runCmdFunc(cmd *cobra.Command, _ []string) error {
	port, err := cmd.Flags().GetString("port")
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %s", err)
	}

	logger := logrus.NewEntry(logrus.New())
	deppyCatsrcAdapter, err := newCatalogSourceAdapter(cmd.Flags(), logger)
	if err != nil {
		return fmt.Errorf("invalid CatalogSourceAdapter configuration: %v", err)
	}

	grpcServer := grpc.NewServer()
	api.RegisterDeppySourceAdapterServer(grpcServer, deppyCatsrcAdapter)
	reflection.Register(grpcServer)

	return graceful.Shutdown(logger, func() error {
		logger.Info("Starting server on port ", port)
		return grpcServer.Serve(lis)
	}, func() {
		grpcServer.GracefulStop()
	})
}

func newCatalogSourceAdapter(flags *pflag.FlagSet, logger *logrus.Entry) (*catalogsource.DeppyAdapter, error) {
	var opts = []catalogsource.AdapterOptions{}
	ns, err := flags.GetString("namespace")
	if err != nil {
		return nil, err
	}
	name, err := flags.GetString("source")
	if err != nil {
		return nil, err
	}
	if ns != "" && name != "" {
		opts = append(opts, catalogsource.WithNamespacedSource(name, ns))
	} else {
		address, err := flags.GetString("address")
		if err != nil {
			return nil, err
		}
		if address != "" {
			opts = append(opts, catalogsource.WithSourceAddress(name, address))
		}
	}
	if logger != nil {
		opts = append(opts, catalogsource.WithLogger(logger))
	}

	return catalogsource.NewCatalogSourceDeppyAdapter(opts...)
}
