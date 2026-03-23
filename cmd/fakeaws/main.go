package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/altacoda/fakeaws/internal/controlplane"
	"github.com/altacoda/fakeaws/internal/engine"
)

var rootCmd = &cobra.Command{
	Use:   "fakeaws",
	Short: "Mock AWS server for SES v2 and STS",
	RunE:  run,
}

func init() {
	rootCmd.Flags().Int("port", 4579, "port to listen on")
	rootCmd.Flags().String("config", "", "config file (TOML)")

	viper.BindPFlag("port", rootCmd.Flags().Lookup("port"))
	viper.BindPFlag("config", rootCmd.Flags().Lookup("config"))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Load config file if specified
	configFile := viper.GetString("config")
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			logger.Error("Failed to read config file", zap.String("file", configFile), zap.Error(err))
			return fmt.Errorf("failed to read config file %s: %w", configFile, err)
		}
		logger.Info("Loaded config file", zap.String("file", configFile))
	}

	// Create engine
	e := engine.NewEngine(logger)

	// Create control plane
	cp := controlplane.New(e, logger)

	// Apply startup config
	if err := applyStartupConfig(cp, e, logger); err != nil {
		return err
	}

	// Mount routes
	mux := http.NewServeMux()
	mux.Handle("/_control/", cp.Handler())
	mux.Handle("/", e)

	port := viper.GetInt("port")
	addr := fmt.Sprintf(":%d", port)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		logger.Info("Shutting down...")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutCtx)
	}()

	// Startup banner
	fmt.Printf("\n  fakeaws listening on %s\n", addr)
	fmt.Printf("    AWS endpoint:   http://localhost:%d\n", port)
	fmt.Printf("    Control plane:  http://localhost:%d/_control/dashboard\n\n", port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func applyStartupConfig(cp *controlplane.ControlPlane, e *engine.Engine, logger *zap.Logger) error {
	// Apply preset if configured
	preset := viper.GetString("preset")
	if preset != "" {
		var configJSON json.RawMessage
		if presetConfig := viper.Get("preset_config"); presetConfig != nil {
			data, err := json.Marshal(presetConfig)
			if err != nil {
				return fmt.Errorf("failed to marshal preset_config: %w", err)
			}
			configJSON = data
		}
		if err := cp.ApplyPreset(preset, configJSON); err != nil {
			return fmt.Errorf("failed to apply preset %q: %w", preset, err)
		}
		logger.Info("Applied startup preset", zap.String("preset", preset))
	}

	// Apply scenarios from config
	var scenarios []controlplane.ScenarioInput
	if err := viper.UnmarshalKey("scenarios", &scenarios); err != nil {
		// Not set or not valid — skip
		return nil
	}
	for _, input := range scenarios {
		s, err := controlplane.ScenarioFromJSON(input)
		if err != nil {
			return fmt.Errorf("failed to create scenario %q: %w", input.Name, err)
		}
		id := e.AddScenario(s)
		logger.Info("Loaded scenario from config", zap.String("name", input.Name), zap.String("id", id))
	}

	return nil
}
