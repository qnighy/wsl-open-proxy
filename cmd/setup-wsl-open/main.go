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
	"strings"

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

type mimeEntry = struct {
	extension string
	mimeTypes []string
}

var mediaGroups = map[string][]mimeEntry{
	"html": {
		{".html", []string{"text/html", "x-scheme-handler/unknown", "x-scheme-handler/about", "x-scheme-handler/https", "x-scheme-handler/http"}},
	},
	"pdf": {
		{".pdf", []string{"application/pdf"}},
	},
	"image": {
		{".png", []string{"image/png"}},
		{".jpg", []string{"image/jpeg"}},
		{".gif", []string{"image/gif"}},
		{".bmp", []string{"image/bmp"}},
		{".svg", []string{"image/svg+xml"}},
	},
	"audio": {
		{".mp3", []string{"audio/mpeg"}},
		{".wav", []string{"audio/wav"}},
		{".ogg", []string{"audio/ogg"}},
	},
	"video": {
		{".mp4", []string{"video/mp4"}},
		{".webm", []string{"video/webm"}},
		{".ogv", []string{"video/ogg"}},
	},
}

func main() {
	updateBin := false
	mediaGroupName := "html"
	var rootCmd = &cobra.Command{
		Use:     "setup-wsl-open",
		Version: wslopenproxy.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errors.New("too many arguments")
			}
			_, ok := mediaGroups[mediaGroupName]
			if !ok {
				return errors.Errorf("Unknown media group: %s", mediaGroupName)
			}
			cmd.SilenceUsage = true
			return run(cmd.Context(), updateBin, mediaGroupName)
		},
	}
	mediaGroupNames := make([]string, 0, len(mediaGroups))
	for name := range mediaGroups {
		mediaGroupNames = append(mediaGroupNames, name)
	}
	rootCmd.Flags().BoolVarP(&updateBin, "update", "u", updateBin, "Update wsl-open-proxy.exe even if it is already installed")
	rootCmd.Flags().StringVarP(&mediaGroupName, "type", "t", mediaGroupName, fmt.Sprintf("Media group to install (One of: %v)", mediaGroupNames))

	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, updateBin bool, mediaGroupName string) error {
	_ = ctx

	exeInstallPath := path.Join(xdg.BinHome, "wsl-open-proxy.exe")
	installBin := updateBin
	_, err := os.Stat(exeInstallPath)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to check existence of wsl-open-proxy.exe")
	} else if err != nil && os.IsNotExist(err) {
		installBin = true
	}

	if installBin {
		exeFile, err := assets.ReadFile(fmt.Sprintf("assets/wsl-open-proxy-%s.exe", runtime.GOARCH))
		if err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "failed to read wsl-open-proxy.exe in assets")
		} else if err != nil && os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Building wsl-open-proxy.exe from source...\n")
			// Not found; alternative installation
			cmd := exec.Command("go", "install", fmt.Sprintf("github.com/qnighy/wsl-open-proxy/cmd/wsl-open-proxy@v%s", wslopenproxy.Version))
			cmd.Env = append(os.Environ(), "GOOS=windows", fmt.Sprintf("GOARCH=%s", runtime.GOARCH))
			_, err := cmd.Output()
			if err != nil {
				return errors.Wrap(err, "failed to build wsl-open-proxy.exe from source")
			}

			fmt.Fprintf(os.Stderr, "Installing wsl-open-proxy.exe built from source...\n")
			gobin := os.Getenv("GOBIN")
			if gobin == "" {
				gopath := os.Getenv("GOPATH")
				if gopath != "" {
					gobin = path.Join(os.Getenv("GOPATH"), "bin")
				} else {
					gobin = path.Join(os.Getenv("HOME"), "go", "bin")
				}
			}
			gobinSuffixed := path.Join(gobin, fmt.Sprintf("windows_%s", runtime.GOARCH))
			exeBuiltPath := path.Join(gobinSuffixed, "wsl-open-proxy.exe")
			if err := os.Rename(exeBuiltPath, exeInstallPath); err != nil {
				return errors.Wrap(err, "failed to move wsl-open-proxy.exe")
			}
		} else {
			fmt.Fprintf(os.Stderr, "Installing prebuilt wsl-open-proxy.exe...\n")
			if err := os.WriteFile(exeInstallPath, exeFile, 0755); err != nil {
				return errors.Wrap(err, "failed to write wsl-open-proxy.exe")
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "wsl-open-proxy.exe is already installed\n")
	}

	mediaGroup, ok := mediaGroups[mediaGroupName]
	if !ok {
		return errors.Errorf("Unknown media group: %s", mediaGroupName)
	}

	fmt.Fprintf(os.Stderr, "Registering desktop entries for %s files...\n", mediaGroupName)
	for _, mimeEntry := range mediaGroup {
		desktopEntry := &xdgini.Config{
			Groups: map[string]*xdgini.ConfigGroup{
				"Desktop Entry": {
					Raws: xdgini.WithOrder(1),
					Entries: map[string]*xdgini.ConfigEntry{
						"Type":      xdgini.OrderedValue("Application", 1),
						"Version":   xdgini.OrderedValue(wslopenproxy.Version, 2),
						"Name":      xdgini.OrderedValue(fmt.Sprintf("WSL Open Proxy (%s)", mimeEntry.extension), 3),
						"NoDisplay": xdgini.OrderedValue("true", 4),
						"Exec":      xdgini.OrderedValue(fmt.Sprintf("wsl-open-proxy.exe --ext %s %%f", mimeEntry.extension), 5),
						"MimeType":  xdgini.OrderedValue(strings.Join(mimeEntry.mimeTypes, ";"), 6),
					},
				},
			},
		}
		extensionName := strings.TrimPrefix(mimeEntry.extension, ".")
		if err := writeFileWithConfirmation(
			path.Join(xdg.DataHome, "applications", fmt.Sprintf("wsl-open-proxy-%s.desktop", extensionName)),
			[]byte(desktopEntry.String()),
			colored(os.Stderr),
		); err != nil {
			return errors.Wrap(err, "failed to write application config")
		}
	}

	fmt.Fprintf(os.Stderr, "Registering mime associations...\n")
	mimeAppsListPath := path.Join(xdg.ConfigHome, "mimeapps.list")
	mimeAppsListText, err := os.ReadFile(mimeAppsListPath)
	if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to read mimeapps.list")
	} else if err != nil && os.IsNotExist(err) {
		mimeAppsListText = []byte{}
	}
	mimeAppsList := xdgini.ParseConfig(string(mimeAppsListText))
	defaultApplications := mimeAppsList.CreateGroup("Default Applications")
	for _, mimeEntry := range mediaGroup {
		extensionName := strings.TrimPrefix(mimeEntry.extension, ".")
		for _, mimeType := range mimeEntry.mimeTypes {
			defaultApplications.CreateEntry(mimeType, fmt.Sprintf("wsl-open-proxy-%s.desktop", extensionName))
		}
	}
	if err := writeFileWithConfirmation(
		mimeAppsListPath,
		[]byte(mimeAppsList.String()),
		colored(os.Stderr),
	); err != nil {
		return errors.Wrap(err, "failed to mime association file")
	}
	fmt.Fprintf(os.Stderr, "Done\n")
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
