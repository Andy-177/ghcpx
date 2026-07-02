// Package cmd parses command line args and runs the corresponding use-case.
package cmd

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/google/wire"
	"github.com/int128/ghcp/pkg/env"
	"github.com/int128/ghcp/pkg/github/client"
	"github.com/int128/ghcp/pkg/password"
	"github.com/int128/ghcp/pkg/usecases/commit"
	"github.com/int128/ghcp/pkg/usecases/forkcommit"
	"github.com/int128/ghcp/pkg/usecases/pullrequest"
	"github.com/int128/ghcp/pkg/usecases/release"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	envGitHubToken = "GITHUB_TOKEN"
	envGitHubAPI   = "GITHUB_API"

	exitCodeOK    = 0
	exitCodeError = 1

	commitCmdName      = "commit"
	emptyCommitCmdName = "empty-commit"
	forkCommitCmdName  = "fork-commit"
	pullRequestCmdName = "pull-request"
	releaseCmdName     = "release"
)

var knownCommandNames = []string{
	commitCmdName,
	emptyCommitCmdName,
	forkCommitCmdName,
	pullRequestCmdName,
	releaseCmdName,
}

var Set = wire.NewSet(
	wire.Bind(new(Interface), new(*Runner)),
	wire.Struct(new(Runner), "Env", "NewGitHub", "NewInternalRunner"),
	wire.Struct(new(InternalRunner), "*"),
)

type Interface interface {
	Run(args []string, version string) int
}

// Runner is the entry point for the command line application.
// It bootstraps the InternalRunner and runs the specified use-case.
type Runner struct {
	Env               env.Interface
	NewGitHub         client.NewFunc
	NewInternalRunner NewInternalRunnerFunc
	password          string
}

// Run parses the command line args and runs the corresponding use-case.
func (r *Runner) Run(args []string, version string) int {
	ctx := context.Background()

	if len(args) > 1 {
		switch args[1] {
		case "password", "passwd":
			return r.runPasswordManagement(args[2:])
		}

		if isKnownSubcommand(args[1]) {
			if !hasTokenFlag(args[2:]) {
				fmt.Fprintln(os.Stderr, "Error: password required")
				fmt.Fprintf(os.Stderr, "Usage: ghcp <password> %s ...\n", args[1])
				return exitCodeError
			}
		} else {
			r.password = args[1]
			args = append([]string{args[0]}, args[2:]...)

			if len(args) > 1 && args[1] == "token" {
				return r.runTokenCommand(args[2:])
			}
		}
	}

	var o globalOptions
	rootCmd := r.newRootCmd(&o)
	commitCmd := r.newCommitCmd(ctx, &o)
	rootCmd.AddCommand(commitCmd)
	emptyCommitCmd := r.newEmptyCommitCmd(ctx, &o)
	rootCmd.AddCommand(emptyCommitCmd)
	forkCommitCmd := r.newForkCommitCmd(ctx, &o)
	rootCmd.AddCommand(forkCommitCmd)
	pullRequestCmd := r.newPullRequestCmd(ctx, &o)
	rootCmd.AddCommand(pullRequestCmd)
	releaseCmd := r.newReleaseCmd(ctx, &o)
	rootCmd.AddCommand(releaseCmd)

	rootCmd.Version = version
	rootCmd.SetArgs(args[1:])
	if err := rootCmd.Execute(); err != nil {
		return exitCodeError
	}
	return exitCodeOK
}

func isKnownSubcommand(name string) bool {
	for _, cmd := range knownCommandNames {
		if cmd == name {
			return true
		}
	}
	return false
}

func hasTokenFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--token" {
			return true
		}
	}
	return false
}

type globalOptions struct {
	Chdir       string
	GitHubToken string
	GitHubAPI   string // optional
	Debug       bool
}

func (o *globalOptions) register(f *pflag.FlagSet) {
	f.StringVarP(&o.Chdir, "directory", "C", "", "Change to directory before operation")
	f.StringVar(&o.GitHubToken, "token", "", fmt.Sprintf("GitHub API token [$%s]", envGitHubToken))
	f.StringVar(&o.GitHubAPI, "api", "", fmt.Sprintf("GitHub API v3 URL (v4 will be inferred) [$%s]", envGitHubAPI))
	f.BoolVar(&o.Debug, "debug", false, "Show debug logs")
}

func (r *Runner) newRootCmd(o *globalOptions) *cobra.Command {
	c := &cobra.Command{
		Use:          "ghcp",
		Short:        "A command to commit files to a GitHub repository",
		SilenceUsage: true,
	}
	o.register(c.PersistentFlags())
	return c
}

func (r *Runner) runPasswordManagement(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  ghcp passwd set <password>            Set password")
		fmt.Fprintln(os.Stderr, "  ghcp passwd change <old> <new>        Change password")
		return exitCodeError
	}
	switch args[0] {
	case "set":
		return r.runPasswordSet(args[1:])
	case "change":
		return r.runPasswordChange(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown passwd command %q\n", args[0])
		return exitCodeError
	}
}

func (r *Runner) runPasswordSet(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: ghcp passwd set <password>")
		return exitCodeError
	}
	hash := password.HashPassword(args[0])
	cfg := &password.Config{Password: hash}
	if err := password.SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not save ghcp.json: %v\n", err)
		return exitCodeError
	}
	fmt.Println("Password set successfully")
	return exitCodeOK
}

func (r *Runner) runPasswordChange(args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: ghcp passwd change <old_password> <new_password>")
		return exitCodeError
	}
	cfg, err := password.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load ghcp.json: %v\n", err)
		return exitCodeError
	}
	if !password.VerifyPassword(cfg.Password, args[0]) {
		fmt.Fprintln(os.Stderr, "Error: incorrect password")
		return exitCodeError
	}
	newHash := password.HashPassword(args[1])
	if cfg.GitHubToken != "" {
		token, err := password.DecryptToken(cfg.GitHubToken, args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not decrypt token: %v\n", err)
			return exitCodeError
		}
		encrypted, err := password.EncryptToken(token, args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not encrypt token: %v\n", err)
			return exitCodeError
		}
		cfg.GitHubToken = encrypted
	}
	cfg.Password = newHash
	if err := password.SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not save ghcp.json: %v\n", err)
		return exitCodeError
	}
	fmt.Println("Password changed successfully")
	return exitCodeOK
}

func (r *Runner) runTokenCommand(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: ghcp <password> token <github_token>")
		return exitCodeError
	}
	cfg, err := password.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load ghcp.json: %v\n", err)
		return exitCodeError
	}
	if !password.VerifyPassword(cfg.Password, r.password) {
		fmt.Fprintln(os.Stderr, "Error: incorrect password")
		return exitCodeError
	}
	encrypted, err := password.EncryptToken(args[0], r.password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not encrypt token: %v\n", err)
		return exitCodeError
	}
	cfg.GitHubToken = encrypted
	if err := password.SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not save ghcp.json: %v\n", err)
		return exitCodeError
	}
	fmt.Println("Token stored successfully")
	return exitCodeOK
}

type NewInternalRunnerFunc func(client.Interface) *InternalRunner

// InternalRunner has the set of use-cases.
type InternalRunner struct {
	CommitUseCase      commit.Interface
	ForkCommitUseCase  forkcommit.Interface
	PullRequestUseCase pullrequest.Interface
	ReleaseUseCase     release.Interface
}

func (r *Runner) newInternalRunner(o *globalOptions) (*InternalRunner, error) {
	log.SetFlags(log.Lmicroseconds)
	if o.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	if o.Chdir != "" {
		if err := r.Env.Chdir(o.Chdir); err != nil {
			return nil, fmt.Errorf("could not change to directory %s: %w", o.Chdir, err)
		}
		slog.Info("Changed to directory", "directory", o.Chdir)
	}
	if o.GitHubToken == "" {
		if r.password == "" {
			return nil, fmt.Errorf("no GitHub API token. Provide password or set --token option")
		}
		cfg, err := password.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("could not load ghcp.json: %w", err)
		}
		if cfg.GitHubToken == "" {
			return nil, fmt.Errorf("no token in ghcp.json. Run 'ghcp <password> token <token>' first")
		}
		token, err := password.DecryptToken(cfg.GitHubToken, r.password)
		if err != nil {
			return nil, fmt.Errorf("could not decrypt token (wrong password?): %w", err)
		}
		o.GitHubToken = token
	}
	if o.GitHubToken == "" {
		return nil, fmt.Errorf("no GitHub API token. Provide password or set --token option")
	}
	if o.GitHubAPI == "" {
		o.GitHubAPI = r.Env.Getenv(envGitHubAPI)
		if o.GitHubAPI != "" {
			slog.Debug("Using GitHub Enterprise URL from environment variable", "variable", envGitHubAPI)
		}
	}
	gh, err := r.NewGitHub(client.Option{
		Token: o.GitHubToken,
		URLv3: o.GitHubAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("could not connect to GitHub API: %w", err)
	}
	return r.NewInternalRunner(gh), nil
}
