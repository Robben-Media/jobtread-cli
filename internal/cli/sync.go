package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Robben-Media/jobtread-cli/internal/catalog"
)

const manifestPath = "internal/catalog/api_manifest.json"

func newSyncCmd(options *globalOptions) *cobra.Command {
	var (
		source       string
		kind         string
		outputPath   string
		validatePath string
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Import and validate authenticated JobTread API coverage",
	}

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import JobTread API coverage into the manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			importKind := strings.TrimSpace(kind)
			if importKind == "" {
				importKind = "auto"
			}

			manifest, err := importManifest(context.Background(), options, source, importKind)
			if err != nil {
				return err
			}

			if outputPath == "" {
				outputPath = manifestPath
			}
			if err := catalog.Save(outputPath, manifest); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "imported %d operations into %s\n", len(manifest.Operations), outputPath)
			return nil
		},
	}
	importCmd.Flags().StringVar(&source, "source", "", "Import source file or URL. Leave empty for the live JobTread Pave schema")
	importCmd.Flags().StringVar(&kind, "kind", "auto", "Import kind: auto|pave|openapi")
	importCmd.Flags().StringVar(&outputPath, "manifest", manifestPath, "Manifest output path")

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate manifest completeness and structure",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := validatePath
			if path == "" {
				path = manifestPath
			}

			manifest, err := catalog.LoadFile(path)
			if err != nil {
				return err
			}

			if err := manifest.Validate(); err != nil {
				return err
			}

			if !manifest.IsPopulated() {
				return fmt.Errorf("manifest has no operations; run `jobtread sync import` to import the live JobTread Pave schema")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "validated %d operations from %s\n", len(manifest.Operations), manifest.Source.URI)
			return nil
		},
	}
	validateCmd.Flags().StringVar(&validatePath, "manifest", manifestPath, "Manifest path to validate")

	cmd.AddCommand(importCmd)
	cmd.AddCommand(validateCmd)
	return cmd
}

func importManifest(ctx context.Context, options *globalOptions, source string, kind string) (catalog.Manifest, error) {
	cfg := options.config()

	switch kind {
	case "pave":
		raw, sourceURI, err := catalog.FetchPaveSchema(ctx, source)
		if err != nil {
			return catalog.Manifest{}, err
		}
		return catalog.ImportPaveSchema(raw, sourceURI)
	case "openapi":
		raw, sourceURI, err := catalog.ReadSource(ctx, source, cfg.Token, cfg.AuthScheme)
		if err != nil {
			return catalog.Manifest{}, err
		}
		return catalog.ImportOpenAPI(raw, sourceURI)
	case "auto":
		if strings.TrimSpace(source) == "" || strings.EqualFold(strings.TrimSpace(source), "jobtread") || strings.EqualFold(strings.TrimSpace(source), "pave") {
			raw, sourceURI, err := catalog.FetchPaveSchema(ctx, "")
			if err != nil {
				return catalog.Manifest{}, err
			}
			return catalog.ImportPaveSchema(raw, sourceURI)
		}

		raw, sourceURI, err := catalog.ReadSource(ctx, source, cfg.Token, cfg.AuthScheme)
		if err != nil {
			return catalog.Manifest{}, err
		}

		if manifest, err := catalog.ImportPaveSchema(raw, sourceURI); err == nil && manifest.IsPopulated() {
			return manifest, nil
		}
		return catalog.ImportOpenAPI(raw, sourceURI)
	default:
		return catalog.Manifest{}, fmt.Errorf("unsupported import kind %q", kind)
	}
}
