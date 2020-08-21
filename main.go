//go:generate go run pkg/codegen/cleanup/main.go
//go:generate /bin/rm -rf pkg/generated
//go:generate go run pkg/codegen/main.go

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	cisoperatorapiv1 "github.com/rancher/clusterscan-operator/pkg/apis/securityscan.cattle.io/v1"
	clusterscan_operator "github.com/rancher/clusterscan-operator/pkg/securityscan"
)

var (
	Version              = "v0.0.0-dev"
	GitCommit            = "HEAD"
	kubeConfig           string
	threads              int
	name                 string
	securityScanImage    = "prachidamle/security-scan"
	securityScanImageTag = "v0.1.20"
	sonobuoyImage        = "rancher/sonobuoy-sonobuoy"
	sonobuoyImageTag     = "v0.16.3"
)

func main() {
	app := cli.NewApp()
	app.Name = "clusterscan-operator"
	app.Version = fmt.Sprintf("%s (%s)", Version, GitCommit)
	app.Usage = "clusterscan-operator needs help!"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "kubeconfig",
			EnvVar:      "KUBECONFIG",
			Destination: &kubeConfig,
		},
		cli.IntFlag{
			Name:        "threads",
			EnvVar:      "CLUSTER_SCAN_OPERATOR_THREADS",
			Value:       2,
			Destination: &threads,
		},
		cli.StringFlag{
			Name:        "name",
			EnvVar:      "CLUSTER_SCAN_OPERATOR_NAME",
			Value:       "clusterscan-operator",
			Destination: &name,
		},
		cli.StringFlag{
			Name:        "security-scan-image",
			EnvVar:      "SECURITY_SCAN_IMAGE",
			Value:       "rancher/security-scan",
			Destination: &securityScanImage,
		},
		cli.StringFlag{
			Name:        "security-scan-image-tag",
			EnvVar:      "SECURITY_SCAN_IMAGE_TAG",
			Value:       "latest",
			Destination: &securityScanImageTag,
		},
		cli.StringFlag{
			Name:        "sonobuoy-image",
			EnvVar:      "SONOBUOY_IMAGE",
			Value:       "rancher/sonobuoy-sonobuoy",
			Destination: &sonobuoyImage,
		},
		cli.StringFlag{
			Name:        "sonobuoy-image-tag",
			EnvVar:      "SONOBUOY_IMAGE_TAG",
			Value:       "v0.16.3",
			Destination: &sonobuoyImageTag,
		},
	}
	app.Action = run

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func run(c *cli.Context) {

	logrus.Info("Starting ClusterScan-Operator")
	ctx := signals.SetupSignalHandler(context.Background())

	kubeConfig = c.String("kubeconfig")
	threads = c.Int("threads")
	securityScanImage = c.String("security-scan-image")
	securityScanImageTag = c.String("security-scan-image-tag")
	sonobuoyImage = c.String("sonobuoy-image")
	sonobuoyImageTag = c.String("sonobuoy-image-tag")
	name = c.String("name")

	kubeConfig, err := kubeconfig.GetNonInteractiveClientConfig(kubeConfig).ClientConfig()
	if err != nil {
		logrus.Fatalf("failed to find kubeconfig: %v", err)
	}

	imgConfig := &cisoperatorapiv1.ScanImageConfig{
		SecurityScanImage:    securityScanImage,
		SecurityScanImageTag: securityScanImageTag,
		SonobuoyImage:        sonobuoyImage,
		SonobuoyImageTag:     sonobuoyImageTag,
	}

	if err := validateConfig(imgConfig); err != nil {
		logrus.Fatalf("Error starting ClusterScan-Operator: %v", err)
	}

	ctl, err := clusterscan_operator.NewController(ctx, kubeConfig, cisoperatorapiv1.ClusterScanNS, name, imgConfig)
	if err != nil {
		logrus.Fatalf("Error building controller: %s", err.Error())
	}
	logrus.Info("Registering ClusterScan controller")

	if err := ctl.Start(ctx, threads, 2*time.Hour); err != nil {
		logrus.Fatalf("Error starting: %v", err)
	}
	<-ctx.Done()
	logrus.Info("Registered ClusterScan controller")
}

func validateConfig(imgConfig *cisoperatorapiv1.ScanImageConfig) error {
	if imgConfig.SecurityScanImage == "" {
		return errors.New("No Security-Scan Image specified")
	}

	if imgConfig.SonobuoyImage == "" {
		return errors.New("No Sonobuoy tool Image specified")
	}

	return nil
}
