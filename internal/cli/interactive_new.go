package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"golang.org/x/term"

	"github.com/clin211/lin/internal/pkg/errs"
	"github.com/clin211/lin/internal/scaffold"
)

func stdinIsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func surveyInterrupt(err error) error {
	if err == terminal.InterruptErr {
		return errs.New(errs.CodeUserCancelled, "cancelled by user (Ctrl+C)")
	}
	return err
}

func gitHubUserForSuggest() string {
	out, err := exec.Command("git", "config", "--get", "github.user").Output()
	if err == nil {
		u := strings.TrimSpace(string(out))
		if u != "" {
			return u
		}
	}
	return "me"
}

func suggestModulePath(projectName string) string {
	base := strings.ToLower(strings.TrimSpace(projectName))
	base = strings.ReplaceAll(base, "-", "_")
	return fmt.Sprintf("github.com/%s/%s", gitHubUserForSuggest(), base)
}

func runNewInteractiveWizard(g *Globals, dryRun bool) (projectName, module, storage, cache, outputDir string, features []string, err error) {
	surveyOpts := []survey.AskOpt{
		survey.WithStdio(os.Stdin, os.Stdout, os.Stderr),
	}

	fmt.Fprintln(os.Stdout, "✨ linctl — create a new project (interactive)")
	fmt.Fprintln(os.Stdout, "")

	if err := survey.AskOne(&survey.Input{
		Message: "Project directory name:",
		Default: "myapp",
		Suggest: func(toComplete string) []string {
			return []string{"myapp", "myblog", "api-server"}
		},
	}, &projectName, append(surveyOpts, survey.WithValidator(func(ans interface{}) error {
		s, ok := ans.(string)
		if !ok {
			return fmt.Errorf("invalid input")
		}
		return scaffold.ValidateProjectName(strings.TrimSpace(s))
	}))...); err != nil {
		return "", "", "", "", "", nil, surveyInterrupt(err)
	}
	projectName = strings.TrimSpace(projectName)

	modDefault := suggestModulePath(projectName)
	if err := survey.AskOne(&survey.Input{
		Message: "Go module path:",
		Default: modDefault,
	}, &module, append(surveyOpts, survey.WithValidator(func(ans interface{}) error {
		s, ok := ans.(string)
		if !ok {
			return fmt.Errorf("invalid input")
		}
		return scaffold.ValidateModulePath(strings.TrimSpace(s))
	}))...); err != nil {
		return "", "", "", "", "", nil, surveyInterrupt(err)
	}
	module = strings.TrimSpace(module)

	storageChoices := []string{"memory", "gorm-sqlite", "gorm-postgres", "gorm-mysql", "mongo"}
	if err := survey.AskOne(&survey.Select{
		Message: "Storage backend:",
		Options: storageChoices,
		Default: "memory",
	}, &storage, surveyOpts...); err != nil {
		return "", "", "", "", "", nil, surveyInterrupt(err)
	}

	cacheChoices := []string{"none", "redis", "bigcache"}
	if err := survey.AskOne(&survey.Select{
		Message: "Cache backend (Redis for shared cache, BigCache for in-process):",
		Options: cacheChoices,
		Default: "none",
	}, &cache, surveyOpts...); err != nil {
		return "", "", "", "", "", nil, surveyInterrupt(err)
	}

	featureChoices := []string{"healthz", "otel", "user", "swagger", "preloader"}
	if err := survey.AskOne(&survey.MultiSelect{
		Message:  "Optional features (space to toggle, enter to confirm):",
		Options:  featureChoices,
		Default:  []string{"healthz"},
		PageSize: 8,
	}, &features, surveyOpts...); err != nil {
		return "", "", "", "", "", nil, surveyInterrupt(err)
	}

	if err := survey.AskOne(&survey.Input{
		Message: "Parent directory for the project (folder will be created inside):",
		Default: ".",
	}, &outputDir, append(surveyOpts, survey.WithValidator(func(ans interface{}) error {
		s, ok := ans.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return fmt.Errorf("output directory is required")
		}
		return nil
	}))...); err != nil {
		return "", "", "", "", "", nil, surveyInterrupt(err)
	}
	outputDir = strings.TrimSpace(outputDir)

	if !g.Yes && !dryRun {
		dest := filepath.Join(outputDir, projectName)
		confirm := false
		if err := survey.AskOne(&survey.Confirm{
			Message: fmt.Sprintf("Create project %q at %s ?", projectName, dest),
			Default: true,
		}, &confirm, surveyOpts...); err != nil {
			return "", "", "", "", "", nil, surveyInterrupt(err)
		}
		if !confirm {
			return "", "", "", "", "", nil, errs.New(errs.CodeUserCancelled, "aborted before scaffold")
		}
	}

	return projectName, module, storage, cache, outputDir, features, nil
}

func promptModuleIfNeeded(g *Globals, projectName, module string) (string, error) {
	module = strings.TrimSpace(module)
	if module != "" {
		return module, nil
	}
	if g.NonInteractive || !stdinIsTerminal() {
		return "", errs.New(errs.CodeInvalidArg, "scaffold: --module is required").
			WithHint("example: --module github.com/yourname/" + projectName)
	}

	surveyOpts := []survey.AskOpt{survey.WithStdio(os.Stdin, os.Stdout, os.Stderr)}
	def := suggestModulePath(projectName)
	var ans string
	if err := survey.AskOne(&survey.Input{
		Message: "Go module path:",
		Default: def,
	}, &ans, append(surveyOpts, survey.WithValidator(func(v interface{}) error {
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("invalid input")
		}
		return scaffold.ValidateModulePath(strings.TrimSpace(s))
	}))...); err != nil {
		return "", surveyInterrupt(err)
	}
	return strings.TrimSpace(ans), nil
}
