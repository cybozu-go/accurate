package sub

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/cybozu-go/accurate"
	"github.com/cybozu-go/accurate/pkg/config"
	"github.com/spf13/cobra"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	defaultConfigPath = "/etc/accurate/config.yaml"
	defaultQPS        = 50
)

var options struct {
	configFile       string
	metricsAddr      string
	probeAddr        string
	leaderElectionID string
	webhookAddr      string
	certDir          string
	qps              int
	zapOpts          zap.Options

	webhookAllowCascadingDeletion bool
}

var rootCmd = &cobra.Command{
	Use:     "accurate-controller",
	Version: accurate.Version,
	Short:   "accurate controller",
	Long:    `accurate controller`,

	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		h, p, err := net.SplitHostPort(options.webhookAddr)
		if err != nil {
			return fmt.Errorf("invalid webhook address: %s, %v", options.webhookAddr, err)
		}
		numPort, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("invalid webhook address: %s, %v", options.webhookAddr, err)
		}
		ns := os.Getenv("POD_NAMESPACE")
		if ns == "" {
			return errors.New("no environment variable POD_NAMESPACE")
		}
		return subMain(ns, h, numPort)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	fs := rootCmd.Flags()
	fs.StringVar(&options.configFile, "config-file", defaultConfigPath, "Configuration file path")
	fs.StringVar(&options.metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to")
	fs.StringVar(&options.probeAddr, "health-probe-addr", ":8081", "Listen address for health probes")
	fs.StringVar(&options.leaderElectionID, "leader-election-id", "accurate", "ID for leader election by controller-runtime")
	fs.StringVar(&options.webhookAddr, "webhook-addr", ":9443", "Listen address for the webhook endpoint")
	fs.StringVar(&options.certDir, "cert-dir", "", "webhook certificate directory")
	fs.IntVar(&options.qps, "apiserver-qps-throttle", defaultQPS, "The maximum QPS to the API server.")

	fs.BoolVar(&options.webhookAllowCascadingDeletion, "webhook-allow-cascading-deletion", false, "Set to true to allow cascading deletion of namespaces (namespaces with children)")

	config.DefaultMutableFeatureGate.AddFlag(fs)

	goflags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(goflags)
	options.zapOpts.BindFlags(goflags)

	fs.AddGoFlagSet(goflags)
}
