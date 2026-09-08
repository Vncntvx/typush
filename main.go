package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Vncntvx/typkg/checker"
	"github.com/Vncntvx/typkg/commands"
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
		Use:   "typkg",
		Short: "A simple package manager for Typst",
		Long:  "A simple package manager for Typst",
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
		Short: "Check if the package is valid",
		Long:  "Check the package against Universe submission rules (bundler + package-check, including local Typst compilation). Must be in the package directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checker.RunWith(cwd(), checker.Options{Local: local, NoCompile: noCompile})
		},
	}
	c.Flags().BoolVar(&local, "local", false, "Only check compiler-minimal rules (name/version/entrypoint)")
	c.Flags().BoolVar(&noCompile, "no-compile", false, "Skip the local Typst compiler checks")
	return c
}

func newCleanCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "clean [package]",
		Short: "Clean the existing dev symlinks",
		Long:  "Clean the existing dev symlinks of all packages (or a certain package) in the data directory.",
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
	var check bool
	c := &cobra.Command{
		Use:   "dev",
		Short: "Create a dev symlink",
		Long:  "Creates a symlink to the package directory for template development.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Dev(cwd(), check)
		},
	}
	c.Flags().BoolVar(&check, "check", false, "Check Universe for name/version conflicts before linking")
	return c
}

func newDownloadCmd() *cobra.Command {
	var checkout, namespace string
	c := &cobra.Command{
		Use:   "download <repository>",
		Short: "Download a package from git repository",
		Long:  "Download a package from git repository to a certain (defaults to `@local`) namespace.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Download(args[0], checkout, namespace)
		},
	}
	c.Flags().StringVarP(&checkout, "checkout", "c", "", "Checkout to a specific tag, commit, or branch")
	c.Flags().StringVarP(&namespace, "namespace", "n", "local", "Namespace to install the package to (without the @ prefix)")
	return c
}

func newExcludeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exclude <files...>",
		Short: "Exclude files for the published bundle",
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
		Short: "Install the current package to a certain namespace",
		Long:  "Install the package to a certain namespace. Must be in the package directory.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Install(cwd(), args[0])
		},
	}
}

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login <registry>",
		Short: "Login to the certain registry",
		Long:  "Login to the certain registry. Currently, only the official Universe (GitHub) registry is supported.",
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
		Short: "Publish the package to a certain registry",
		Long:  "Publish the package to a certain registry. Currently, only the official Universe (GitHub) registry is supported.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "universe" {
				return fmt.Errorf("unsupported registry: %s", args[0])
			}
			return commands.Publish(cwd(), dryRun)
		},
	}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run the publish process. No actual changes will be made.")
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

// Back-compat: `typkg host` == `typkg ci plan`
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

// Back-compat: `typkg generate` == `typkg ci generate`
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
