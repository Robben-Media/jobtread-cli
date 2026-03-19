package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Robben-Media/jobtread-cli/internal/config"
	"github.com/Robben-Media/jobtread-cli/internal/httpclient"
	"github.com/Robben-Media/jobtread-cli/internal/output"
	"github.com/Robben-Media/jobtread-cli/internal/pave"
)

type globalOptions struct {
	baseURL         string
	token           string
	authScheme      string
	keychainService string
	timeout         time.Duration
	headerPairs     []string
	outputFormat    string
	query           string
	meta            bool
	debug           bool
	color           string
}

func addPersistentFlags(cmd *cobra.Command, options *globalOptions) {
	cmd.PersistentFlags().StringVar(&options.baseURL, "base-url", "", "Base JobTread API URL")
	cmd.PersistentFlags().StringVar(&options.token, "token", "", "JobTread API token override")
	cmd.PersistentFlags().StringVar(&options.authScheme, "auth-scheme", "", "Authorization scheme override")
	cmd.PersistentFlags().StringVar(&options.keychainService, "keychain-service", "", "Exact macOS keychain service name")
	cmd.PersistentFlags().DurationVar(&options.timeout, "timeout", 0, "HTTP timeout override")
	cmd.PersistentFlags().StringArrayVar(&options.headerPairs, "header", nil, "Header in key=value form")
	cmd.PersistentFlags().StringVar(&options.outputFormat, "output", "auto", "Output format: auto|json|toon|table|raw")
	cmd.PersistentFlags().StringVar(&options.query, "query", "", "JMESPath expression applied before formatting")
	cmd.PersistentFlags().BoolVar(&options.meta, "meta", false, "Print response metadata to stderr")
	cmd.PersistentFlags().BoolVar(&options.debug, "debug", false, "Print request debugging details to stderr")
	cmd.PersistentFlags().StringVar(&options.color, "color", "auto", "Color mode: auto|always|never")
}

func (o globalOptions) config() config.Config {
	cfg := config.Load()
	if strings.TrimSpace(o.baseURL) != "" {
		cfg.BaseURL = strings.TrimSpace(o.baseURL)
	}
	if strings.TrimSpace(o.token) != "" {
		cfg.Token = strings.TrimSpace(o.token)
	}
	if strings.TrimSpace(o.authScheme) != "" {
		cfg.AuthScheme = strings.TrimSpace(o.authScheme)
	}
	if strings.TrimSpace(o.keychainService) != "" {
		cfg.KeychainService = strings.TrimSpace(o.keychainService)
	}
	if o.timeout > 0 {
		cfg.Timeout = o.timeout
	}
	return cfg
}

func (o globalOptions) resolveRequestClient(ctx context.Context) (*httpclient.Client, string, error) {
	cfg := o.config()
	token, tokenSource, err := cfg.ResolveToken(ctx)
	if err != nil {
		return nil, "", err
	}
	cfg.Token = token
	if err := cfg.ValidateRequestAuth(); err != nil {
		return nil, "", err
	}
	return httpclient.New(cfg), tokenSource, nil
}

func (o globalOptions) emit(cmd *cobra.Command, response httpclient.Response) error {
	formatted, actualFormat, err := output.Process(response.Body, output.Format(o.outputFormat), o.query)
	if err != nil {
		return err
	}

	if _, err := cmd.OutOrStdout().Write(formatted); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	if len(formatted) == 0 || formatted[len(formatted)-1] != '\n' {
		if _, err := io.WriteString(cmd.OutOrStdout(), "\n"); err != nil {
			return fmt.Errorf("write output newline: %w", err)
		}
	}

	if o.meta || o.debug {
		meta := map[string]any{
			"status":   response.StatusCode,
			"method":   response.Method,
			"url":      response.URL,
			"duration": response.Duration.String(),
			"format":   actualFormat,
		}
		raw, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("encode metadata: %w", err)
		}
		if _, err := cmd.ErrOrStderr().Write(append(raw, '\n')); err != nil {
			return fmt.Errorf("write metadata: %w", err)
		}
	}

	return nil
}

func (o globalOptions) resolvePaveClient(ctx context.Context, public bool) (*pave.Client, string, error) {
	cfg := o.config()
	tokenSource := "public"
	grantKey := ""
	if !public {
		resolved, source, err := pave.ResolveGrantKey(ctx, cfg)
		if err != nil {
			return nil, "", err
		}
		grantKey = resolved
		tokenSource = source
	}

	return pave.New(cfg.BaseURL, grantKey, cfg.Timeout, ""), tokenSource, nil
}
