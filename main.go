package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Vncntvx/typush/checker"
	"github.com/Vncntvx/typush/commands"
)

var version = "dev"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err := NewRoot().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:          "typush",
		Version:      version,
		Short:        "A package manager for Typst",
		Long:         "typush develops, validates, and publishes Typst packages.",
		SilenceUsage: true,
	}
	root.AddCommand(
		newCheckCmd(),
		newCleanCmd(),
		newDevCmd(),
		newDownloadCmd(),
		newExcludeCmd(),
		newInitCmd(),
		newInstallCmd(),
		newLoginCmd(),
		newPublishCmd(),
		newPRCmd(),
		newBumpCmd(),
		newMetadataCmd(),
		newPathCmd(),
		newCICmd(),
		// Back-compat aliases for the Rust CLI (breaking allowed, but keep them working):
		newHostAliasCmd(),
		newGenerateAliasCmd(),
	)
	return root
}

func cwd() string {
	d, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: failed to get current directory")
		os.Exit(1)
	}
	return d
}

func newCheckCmd() *cobra.Command {
	var local, noCompile bool
	c := &cobra.Command{
		Use:   "check",
		Short: "Validate the package against specification rules",
		Long:  "Validate the package against Universe submission rules and run local compiler checks. Must be run in the package directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checker.RunWith(cwd(), checker.Options{Local: local, NoCompile: noCompile})
		},
	}
	c.Flags().BoolVar(&local, "local", false, "Validate manifest fields only")
	c.Flags().BoolVar(&noCompile, "no-compile", false, "Skip local Typst compiler checks")
	return c
}

func newCleanCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "clean [package]",
		Short: "Remove development symlinks in @preview",
		Long:  "Remove development symlinks for all packages or a specified package in the data directory.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return commands.CleanOne(args[0])
			}
			return commands.CleanAll()
		},
	}
	return c
}

func newDevCmd() *cobra.Command {
	var (
		check    bool
		listFlag bool
	)
	c := &cobra.Command{
		Use:   "dev",
		Short: "Link the package directory into @preview or list active links",
		Long:  "Create a symlink in @preview for package development, or list existing symlinks.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if listFlag {
				return commands.DevList()
			}
			return commands.Dev(cwd(), check)
		},
	}
	c.Flags().BoolVar(&check, "check", false, "Check Universe for remote naming conflicts before linking")
	c.Flags().BoolVarP(&listFlag, "list", "l", false, "List active dev symlinks in @preview")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List active dev symlinks in @preview",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.DevList()
		},
	}
	c.AddCommand(listCmd)
	return c
}

func newDownloadCmd() *cobra.Command {
	var checkout, namespace string
	c := &cobra.Command{
		Use:   "download <repository>",
		Short: "Download a package from a git repository",
		Long:  "Download a package from a git repository into a local namespace (defaults to @local).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Download(args[0], checkout, namespace)
		},
	}
	c.Flags().StringVarP(&checkout, "checkout", "c", "", "Checkout a specific tag, commit, or branch")
	c.Flags().StringVarP(&namespace, "namespace", "n", "local", "Namespace to install the package to (without the @ prefix)")
	return c
}

func newExcludeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exclude <files...>",
		Short: "Exclude files from the published bundle",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Add(cwd(), args)
		},
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize a new package in the current directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			return commands.Init(cwd(), name)
		},
	}
}

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <target>",
		Short: "Install the current package to a namespace",
		Long:  "Install the package in the current directory to a specified namespace.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Install(cwd(), args[0])
		},
	}
}

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login <registry>",
		Short: "Verify authentication for a registry",
		Long:  "Verify GitHub authentication for the Universe registry via the gh CLI.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "universe" {
				return fmt.Errorf("unsupported registry: %s", args[0])
			}
			return commands.Login()
		},
	}
}

func newPublishCmd() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "publish <registry>",
		Short: "Publish the package to a registry",
		Long:  "Publish the package to the official Universe registry via the gh CLI.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "universe" {
				return fmt.Errorf("unsupported registry: %s", args[0])
			}
			return commands.Publish(cwd(), dryRun)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate publishing without creating branches or pull requests")
	return c
}

func newCICmd() *cobra.Command {
	c := &cobra.Command{Use: "ci", Short: "CI helpers"}
	var (
		genSource, genPushToFork, genDestination string
		planPackages                             string
	)
	gen := &cobra.Command{
		Use:   "generate",
		Short: "Generate CI that publishes the package automatically",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Generate(cwd(), genSource, genPushToFork, genDestination)
		},
	}
	gen.Flags().StringVar(&genSource, "source", "", "The destination typst/packages")
	gen.Flags().StringVar(&genPushToFork, "push-to-fork", "", "The forked repository from typst/packages")
	gen.Flags().StringVar(&genDestination, "destination", "", "The path to install the package in the source repository")
	plan := &cobra.Command{
		Use:   "plan",
		Short: "Scan workspace and print the CI matrix JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			var only []string
			s := strings.TrimSpace(planPackages)
			if s != "" && s != "all" {
				only = strings.Fields(s)
			}
			return commands.Plan(cwd(), only)
		},
	}
	plan.Flags().StringVar(&planPackages, "packages", "all", "Space-separated list of packages to host, or 'all'")
	c.AddCommand(gen, plan)
	return c
}

// Back-compat: `typush host` == `typush ci plan`
func newHostAliasCmd() *cobra.Command {
	var packages string
	c := &cobra.Command{
		Use:    "host",
		Short:  "Host the package in a GitHub repository (deprecated: use `ci plan`)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var only []string
			s := strings.TrimSpace(packages)
			if s != "" && s != "all" {
				only = strings.Fields(s)
			}
			return commands.Plan(cwd(), only)
		},
	}
	c.Flags().StringVar(&packages, "packages", "all", "Space-separated list of packages to host")
	_ = c.Flags().String("source", "", "")
	_ = c.Flags().String("destination", "", "")
	_ = c.Flags().String("tag", "", "")
	_ = c.Flags().String("output-format", "json", "")
	return c
}

// Back-compat: `typush generate` == `typush ci generate`
func newGenerateAliasCmd() *cobra.Command {
	var source, pushToFork, destination string
	c := &cobra.Command{
		Use:    "generate",
		Short:  "Generate CI (deprecated: use `ci generate`)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Generate(cwd(), source, pushToFork, destination)
		},
	}
	c.Flags().StringVar(&source, "source", "", "")
	c.Flags().StringVar(&pushToFork, "push-to-fork", "", "")
	c.Flags().StringVar(&destination, "destination", "", "")
	return c
}

func newPRCmd() *cobra.Command {
	c := &cobra.Command{
		Use:          "pr",
		Short:        "Manage and inspect Universe pull requests",
		Long:         "Inspect open pull requests and CI check runs on the official Universe repository.",
		SilenceUsage: true,
	}
	status := &cobra.Command{
		Use:   "status [number|url]",
		Short: "View PR details, comments, and reviewer activity",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := ""
			if len(args) == 1 {
				ref = args[0]
			}
			return commands.PRStatus(cwd(), ref)
		},
	}
	var watch bool
	checks := &cobra.Command{
		Use:   "checks [number|url]",
		Short: "View or watch CI check runs for a submission PR",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := ""
			if len(args) == 1 {
				ref = args[0]
			}
			return commands.PRChecks(cwd(), ref, watch)
		},
	}
	checks.Flags().BoolVarP(&watch, "watch", "w", false, "Watch CI checks until they complete")
	c.AddCommand(status, checks)
	return c
}

func newBumpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bump [patch|minor|major|<version>]",
		Short: "Bump package version in typst.toml",
		Long:  "Bump package version in typst.toml (supports patch, minor, major, or explicit version).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			return commands.Bump(cwd(), target)
		},
	}
}

func newMetadataCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "metadata [field]",
		Short: "Display or query package manifest metadata",
		Long:  "Display package metadata in human-readable or JSON format, or query a specific field (e.g. name, version, authors).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			field := ""
			if len(args) == 1 {
				field = args[0]
			}
			return commands.Metadata(cwd(), field, asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "Output metadata in JSON format")
	return c
}

func newPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path [namespace]",
		Short: "Show local Typst package directory path",
		Long:  "Show the local Typst packages directory path, or the path for a specific namespace (e.g. preview, local).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ns := ""
			if len(args) == 1 {
				ns = args[0]
			}
			return commands.Path(ns)
		},
	}
}
