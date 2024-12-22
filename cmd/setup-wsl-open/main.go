package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/adrg/xdg"
	"github.com/c-bata/go-prompt"
	"github.com/pkg/errors"
	wslopenproxy "github.com/qnighy/wsl-open-proxy"
	"github.com/qnighy/wsl-open-proxy/xdgini"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

//go:embed assets/*.keep assets/*
var assets embed.FS

func main() {
	var rootCmd = &cobra.Command{
		Use:     "setup-wsl-open",
		Version: wslopenproxy.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errors.New("too many arguments")
			}
			cmd.SilenceUsage = true
			return run(cmd.Context())
		},
	}

	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	_ = ctx

	exeInstallPath := path.Join(xdg.BinHome, "wsl-open-proxy.exe")
	exeFile, err := assets.ReadFile(fmt.Sprintf("assets/wsl-open-proxy-%s.exe", runtime.GOARCH))
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to read wsl-open-proxy.exe in assets")
	} else if err != nil && os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Building wsl-open-proxy.exe from source...\n")
		// Not found; alternative installation
		cmd := exec.Command("go", "build", "-o", exeInstallPath, "github.com/qnighy/wsl-open-proxy/cmd/wsl-open-proxy")
		// cmd := exec.Command("go", "build", "-o", exeInstallPath, "./cmd/wsl-open-proxy")
		cmd.Env = append(os.Environ(), "GOOS=windows", fmt.Sprintf("GOARCH=%s", runtime.GOARCH))
		_, err := cmd.Output()
		if err != nil {
			return errors.Wrap(err, "failed to build wsl-open-proxy.exe from source")
		}
	} else {
		fmt.Fprintf(os.Stderr, "Installing wsl-open-proxy.exe...\n")
		if err := os.WriteFile(exeInstallPath, exeFile, 0755); err != nil {
			return errors.Wrap(err, "failed to write wsl-open-proxy.exe")
		}
	}

	desktopEntry := &xdgini.Config{
		Groups: map[string]*xdgini.ConfigGroup{
			"Desktop Entry": {
				Raws: xdgini.WithOrder(1),
				Entries: map[string]*xdgini.ConfigEntry{
					"Type":      xdgini.OrderedValue("Application", 1),
					"Version":   xdgini.OrderedValue(wslopenproxy.Version, 2),
					"Name":      xdgini.OrderedValue("WSL Open Proxy (as HTML)", 3),
					"NoDisplay": xdgini.OrderedValue("true", 4),
					"Exec":      xdgini.OrderedValue("wsl-open-proxy.exe --ext .html %f", 5),
					"MimeType":  xdgini.OrderedValue("x-scheme-handler/unknown;x-scheme-handler/about;x-scheme-handler/https;x-scheme-handler/http;text/html;", 6),
				},
			},
		},
	}
	if err := writeFileWithConfirmation(
		path.Join(xdg.DataHome, "applications", "wsl-open-proxy-html.desktop"),
		[]byte(desktopEntry.String()),
		colored(os.Stderr),
	); err != nil {
		return errors.Wrap(err, "failed to write application config")
	}

	mimeAppsListPath := path.Join(xdg.ConfigHome, "mimeapps.list")
	mimeAppsListText, err := os.ReadFile(mimeAppsListPath)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to read mimeapps.list")
	} else if err != nil && os.IsNotExist(err) {
		mimeAppsListText = []byte{}
	}
	mimeAppsList := xdgini.ParseConfig(string(mimeAppsListText))
	defaultApplications := mimeAppsList.CreateGroup("Default Applications")
	defaultApplications.CreateEntry("text/html", "wsl-open-proxy-html.desktop")
	defaultApplications.CreateEntry("x-scheme-handler/http", "wsl-open-proxy-html.desktop")
	defaultApplications.CreateEntry("x-scheme-handler/https", "wsl-open-proxy-html.desktop")
	defaultApplications.CreateEntry("x-scheme-handler/about", "wsl-open-proxy-html.desktop")
	defaultApplications.CreateEntry("x-scheme-handler/unknown", "wsl-open-proxy-html.desktop")
	if err := writeFileWithConfirmation(
		mimeAppsListPath,
		[]byte(mimeAppsList.String()),
		colored(os.Stderr),
	); err != nil {
		return errors.Wrap(err, "failed to mime association file")
	}
	return nil
}

func writeFileWithConfirmation(filePath string, data []byte, coloredStderr bool) error {
	oldContent, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to read %s", filePath)
	} else if err == nil {
		if string(oldContent) == string(data) {
			// No need to update
			return nil
		}
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(oldContent), string(data), false)
		fmt.Fprintf(os.Stderr, "Need to apply the following changes to %s:\n", filePath)
		if coloredStderr {
			fmt.Fprintf(os.Stderr, "%s\n", dmp.DiffPrettyText(diffs))
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", diffText(diffs))
		}
		answer := prompt.Input("Apply this change? [y/N]", yesNoCompleter)
		if answer != "y" && answer != "Y" {
			return errors.Errorf("Overwrite to %s is canceled", filePath)
		}
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return errors.Wrapf(err, "failed to write %s", filePath)
	}
	return nil
}

func diffText(diffs []diffmatchpatch.Diff) string {
	var buff bytes.Buffer
	for _, diff := range diffs {
		text := diff.Text

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			_, _ = buff.WriteString(text)
		case diffmatchpatch.DiffDelete:
			_, _ = buff.WriteString(text)
		case diffmatchpatch.DiffEqual:
			_, _ = buff.WriteString(text)
		}
	}

	return buff.String()
}

func yesNoCompleter(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{
		{Text: "y", Description: "Apply this change"},
		{Text: "n", Description: "Do not apply this change"},
	}
}

func colored(f *os.File) bool {
	return term.IsTerminal(int(f.Fd())) && os.Getenv("NO_COLOR") == ""
}
