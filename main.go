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
		newCloneCmd(),
		newCompletionCmd(),
		newDevCmd(),
		newDownloadCmd(),
		newExcludeCmd(),
		newInitCmd(),
		newInstallCmd(),
		newListCmd(),
		newLoginCmd(),
		newPublishCmd(),
		newPRCmd(),
		newBumpCmd(),
		newMetadataCmd(),
		newPathCmd(),
		newSearchCmd(),
		newInfoCmd(),
		newOutdatedCmd(),
		newUninstallCmd(),
		newUpdateCmd(),
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

// registerDryRun adds the shared -n/--dry-run preview flag to a command.
func registerDryRun(cmd *cobra.Command, dst *bool, usage string) {
	cmd.Flags().BoolVarP(dst, "dry-run", "n", false, usage)
}

// registerDryRunNoShorthand adds --dry-run without a shorthand, for commands
// that already use -n for something else (download's --namespace).
func registerDryRunNoShorthand(cmd *cobra.Command, dst *bool, usage string) {
	cmd.Flags().BoolVar(dst, "dry-run", false, usage)
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
	var dryRun bool
	c := &cobra.Command{
		Use:   "clean [package]",
		Short: "Remove development symlinks in @preview",
		Long:  "Remove development symlinks for all packages or a specified package in the data directory.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return commands.CleanOne(args[0], dryRun)
			}
			return commands.CleanAll(dryRun)
		},
	}
	registerDryRun(c, &dryRun, "Preview symlinks to be removed without deleting them")
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
	var (
		checkout, namespace, subdir string
		dryRun                      bool
	)
	c := &cobra.Command{
		Use:   "download <repository>",
		Short: "Download a package from a git repository",
		Long:  "Download a package from a git repository into a local namespace (defaults to @local).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Download(commands.DownloadOptions{
				Repository: args[0],
				Checkout:   checkout,
				Namespace:  namespace,
				Subdir:     subdir,
				DryRun:     dryRun,
			})
		},
	}
	c.Flags().StringVarP(&checkout, "checkout", "c", "", "Checkout a specific tag, commit, or branch")
	c.Flags().StringVarP(&namespace, "namespace", "n", "local", "Namespace to install the package to (without the @ prefix)")
	c.Flags().StringVar(&subdir, "subdir", "", "Subdirectory inside the repository to install")
	registerDryRunNoShorthand(c, &dryRun, "Clone and preview the installation without copying files into the namespace")
	return c
}

func newExcludeCmd() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{
		Use:   "exclude <files...>",
		Short: "Exclude files from the published bundle",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Add(cwd(), args, dryRun)
		},
	}
	registerDryRun(c, &dryRun, "Preview exclude patterns to add without updating typst.toml")
	return c
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
	var dryRun bool
	c := &cobra.Command{
		Use:   "install <target>",
		Short: "Install the current package to a namespace",
		Long:  "Install the package in the current directory to a specified namespace.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Install(cwd(), args[0], dryRun)
		},
	}
	registerDryRun(c, &dryRun, "Preview files to be installed without writing to disk")
	return c
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
	registerDryRun(c, &dryRun, "Simulate publishing without creating branches or pull requests")
	return c
}

func newCICmd() *cobra.Command {
	c := &cobra.Command{Use: "ci", Short: "CI helpers"}
	var (
		genSource, genPushToFork, genDestination string
		genDryRun                                bool
		planPackages                             string
	)
	gen := &cobra.Command{
		Use:   "generate",
		Short: "Generate CI that publishes the package automatically",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Generate(cwd(), commands.GenerateOptions{
				Source:      genSource,
				PushToFork:  genPushToFork,
				Destination: genDestination,
				DryRun:      genDryRun,
			})
		},
	}
	gen.Flags().StringVar(&genSource, "source", "", "The destination typst/packages")
	gen.Flags().StringVar(&genPushToFork, "push-to-fork", "", "The forked repository from typst/packages")
	gen.Flags().StringVar(&genDestination, "destination", "", "The path to install the package in the source repository")
	registerDryRun(gen, &genDryRun, "Print the generated workflow to stdout without creating files")
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
	var dryRun bool
	c := &cobra.Command{
		Use:    "generate",
		Short:  "Generate CI (deprecated: use `ci generate`)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Generate(cwd(), commands.GenerateOptions{
				Source:      source,
				PushToFork:  pushToFork,
				Destination: destination,
				DryRun:      dryRun,
			})
		},
	}
	c.Flags().StringVar(&source, "source", "", "")
	c.Flags().StringVar(&pushToFork, "push-to-fork", "", "")
	c.Flags().StringVar(&destination, "destination", "", "")
	registerDryRun(c, &dryRun, "Print the generated workflow to stdout without creating files")
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
	var opt commands.BumpOptions
	c := &cobra.Command{
		Use:   "bump [patch|minor|major|<version>]",
		Short: "Bump package version in typst.toml",
		Long: `Bump package version in typst.toml (supports patch, minor, major, or explicit version).
Use --include to update version strings in additional files like README.md.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.Dir = cwd()
			if len(args) == 1 {
				opt.Target = args[0]
			}
			return commands.Bump(opt)
		},
	}
	registerDryRun(c, &opt.DryRun, "Preview the version bump without updating files")
	c.Flags().StringSliceVarP(&opt.Include, "include", "i", nil, "Additional files to update version in")
	c.Flags().StringVarP(&opt.Tag, "tag", "t", "", "HTML/XML tag enclosing the version to replace (e.g. 'version')")
	return c
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

func newSearchCmd() *cobra.Command {
	var (
		opt   commands.UniverseOptions
		limit int
	)
	c := &cobra.Command{
		Use:   "search <query>",
		Short: "Search packages in Typst Universe",
		Long:  "Search packages in Typst Universe by name, description, categories, and keywords.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Search(args[0], limit, opt)
		},
	}
	c.Flags().IntVarP(&limit, "limit", "l", commands.DefaultSearchLimit, "Maximum number of search results")
	c.Flags().BoolVar(&opt.AsJSON, "json", false, "Output results in JSON format")
	c.Flags().BoolVar(&opt.Refresh, "refresh", false, "Refresh the local Universe index cache")
	return c
}

func newInfoCmd() *cobra.Command {
	var opt commands.UniverseOptions
	c := &cobra.Command{
		Use:   "info <package>[:version]",
		Short: "Show package details from Typst Universe",
		Long:  "Display detailed metadata, authors, license, compiler requirements, and version history for a package in Typst Universe.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Info(args[0], opt)
		},
	}
	c.Flags().BoolVar(&opt.AsJSON, "json", false, "Output package info in JSON format")
	c.Flags().BoolVar(&opt.Refresh, "refresh", false, "Refresh the local Universe index cache")
	return c
}

func newOutdatedCmd() *cobra.Command {
	var opt commands.UniverseOptions
	c := &cobra.Command{
		Use:   "outdated [path]",
		Short: "Check for outdated package dependencies in .typ files",
		Long:  "Scan .typ files recursively for @preview package imports and compare them against the latest versions in Typst Universe.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := cwd()
			if len(args) == 1 {
				target = args[0]
			}
			return commands.Outdated(target, opt)
		},
	}
	c.Flags().BoolVar(&opt.AsJSON, "json", false, "Output outdated dependencies in JSON format")
	c.Flags().BoolVar(&opt.Refresh, "refresh", false, "Refresh the local Universe index cache")
	return c
}

func newUpdateCmd() *cobra.Command {
	var opt commands.UpdateOptions
	c := &cobra.Command{
		Use:   "update [package...]",
		Short: "Update package dependencies in .typ files to latest versions",
		Long:  "Update @preview package imports in .typ files to their latest versions from Typst Universe.",
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.Dir = cwd()
			opt.Packages = args
			return commands.Update(opt)
		},
	}
	registerDryRun(c, &opt.DryRun, "Preview dependency updates without modifying files")
	c.Flags().StringVarP(&opt.File, "file", "f", "", "Update dependencies only within the specified .typ file")
	c.Flags().BoolVar(&opt.Refresh, "refresh", false, "Refresh the local Universe index cache")
	return c
}

func newListCmd() *cobra.Command {
	var opt commands.ListOptions
	c := &cobra.Command{
		Use:   "list [namespace]",
		Short: "List installed packages",
		Long:  "List packages installed in the local Typst package directory. Use --all to include cached packages.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opt.Namespace = args[0]
			}
			return commands.List(opt)
		},
	}
	c.Flags().BoolVarP(&opt.All, "all", "a", false, "Include packages in the Typst download cache")
	c.Flags().BoolVarP(&opt.Tree, "tree", "t", false, "Display packages as a tree")
	c.Flags().BoolVar(&opt.JSON, "json", false, "Output package list in JSON format")
	return c
}

func newUninstallCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)
	c := &cobra.Command{
		Use:     "uninstall <target>",
		Aliases: []string{"remove", "rm"},
		Short:   "Remove installed packages",
		Long: `Remove packages from the local Typst package directory.

Target syntax:
  @ns/pkg:ver   Remove a specific version
  @ns/pkg       Remove all versions of a package
  @ns           Remove an entire namespace
  pkg:ver       Remove from @local (default namespace)
  pkg           Remove all versions from @local`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.Uninstall(commands.UninstallOptions{
				Target: args[0],
				Force:  force,
				DryRun: dryRun,
			})
		},
	}
	c.Flags().BoolVarP(&force, "force", "y", false, "Skip confirmation prompts")
	registerDryRun(c, &dryRun, "Preview packages to remove without deleting them")
	return c
}

func newCloneCmd() *cobra.Command {
	var (
		force  bool
		dryRun bool
	)
	c := &cobra.Command{
		Use:   "clone <package> [destination]",
		Short: "Download package source from Typst Universe",
		Long:  "Download and extract a package from the Typst Universe CDN. Accepts @preview/name:version, name:version, or name (latest).",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dest := ""
			if len(args) == 2 {
				dest = args[1]
			}
			return commands.Clone(commands.CloneOptions{
				Spec:   args[0],
				Dest:   dest,
				Force:  force,
				DryRun: dryRun,
			})
		},
	}
	c.Flags().BoolVarP(&force, "force", "f", false, "Overwrite non-empty destination directory")
	registerDryRun(c, &dryRun, "Preview the clone without downloading")
	return c
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion <bash|zsh|fish|powershell>",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for typush.

To load completions:

Bash:
  $ source <(typush completion bash)
  # Or add to ~/.bashrc:
  $ typush completion bash > /etc/bash_completion.d/typush

Zsh:
  $ typush completion zsh > "${fpath[1]}/_typush"
  # Then restart your shell.

Fish:
  $ typush completion fish | source
  # Or persist:
  $ typush completion fish > ~/.config/fish/completions/typush.fish

PowerShell:
  PS> typush completion powershell | Out-String | Invoke-Expression`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell %q (supported: bash, zsh, fish, powershell)", args[0])
			}
		},
	}
}
