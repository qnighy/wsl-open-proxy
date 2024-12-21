package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sys/windows"
)

var modShlwapi = windows.NewLazySystemDLL("Shlwapi.dll")
var procAssocQueryStringW = modShlwapi.NewProc("AssocQueryStringW")

const NULL = 0

func main() {
	var ext string
	var rootCmd = &cobra.Command{
		Use: "wsl-open-proxy file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("file is required")
			} else if len(args) > 1 {
				return errors.New("too many arguments")
			}
			cmd.SilenceUsage = true
			return run(cmd.Context(), args[0], ext)
		},
	}

	rootCmd.Flags().StringVar(&ext, "ext", ext, "overrides file extension")

	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, file string, ext string) error {
	_ = ctx
	if ext == "" {
		ext = filepath.Ext(file)
	}
	if ext == "" {
		return errors.New("No file extension found")
	}
	assoc, err := SafeAssocQueryString(ASSOCF_NONE, ASSOCSTR_EXECUTABLE, ext, "open")
	if err != nil {
		return errors.Wrap(err, "error getting executable for file extension")
	}

	wFile, err := wslpath(ctx, file)
	if err != nil {
		return errors.Wrap(err, "error converting file path to Windows absolute path")
	}
	cmd := strings.ReplaceAll(strings.ReplaceAll(assoc, "%1", wFile), "%L", wFile)

	commandLinePtr, err := windows.UTF16PtrFromString(cmd)
	if err != nil {
		return err
	}
	var s windows.StartupInfo
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(nil, commandLinePtr, nil, nil, false, 0, nil, nil, &s, &pi); err != nil {
		return errors.Wrapf(err, "error executing command %#v", cmd)
	}
	return nil
}

func wslpath(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "wsl", "wslpath", "-w", path)
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "error calling wslpath")
	}
	return strings.TrimSpace(string(out)), nil
}

const (
	ASSOCF_NONE                 = 0x00000000
	ASSOCF_INIT_NOREMAPCLSID    = 0x00000001
	ASSOCF_INIT_BYEXENAME       = 0x00000002
	ASSOCF_OPEN_BYEXENAME       = 0x00000002
	ASSOCF_INIT_DEFAULTTOSTAR   = 0x00000004
	ASSOCF_INIT_DEFAULTTOFOLDER = 0x00000008
	ASSOCF_NOUSERSETTINGS       = 0x00000010
	ASSOCF_NOTRUNCATE           = 0x00000020
	ASSOCF_VERIFY               = 0x00000040
	ASSOCF_REMAPRUNDLL          = 0x00000080
	ASSOCF_NOFIXUPS             = 0x00000100
	ASSOCF_IGNOREBASECLASS      = 0x00000200
	ASSOCF_INIT_IGNOREUNKNOWN   = 0x00000400
	ASSOCF_INIT_FIXED_PROGID    = 0x00000800
	ASSOCF_IS_PROTOCOL          = 0x00001000
	ASSOCF_INIT_FOR_FILE        = 0x00002000
	ASSOCF_IS_FULL_URI          = 0x00004000
	ASSOCF_PER_MACHINE_ONLY     = 0x00008000
	ASSOCF_APP_TO_APP           = 0x00010000
)

const (
	ASSOCSTR_COMMAND = 1
	ASSOCSTR_EXECUTABLE
	ASSOCSTR_FRIENDLYDOCNAME
	ASSOCSTR_FRIENDLYAPPNAME
	ASSOCSTR_NOOPEN
	ASSOCSTR_SHELLNEWVALUE
	ASSOCSTR_DDECOMMAND
	ASSOCSTR_DDEIFEXEC
	ASSOCSTR_DDEAPPLICATION
	ASSOCSTR_DDETOPIC
	ASSOCSTR_INFOTIP
	ASSOCSTR_QUICKTIP
	ASSOCSTR_TILEINFO
	ASSOCSTR_CONTENTTYPE
	ASSOCSTR_DEFAULTICON
	ASSOCSTR_SHELLEXTENSION
	ASSOCSTR_DROPTARGET
	ASSOCSTR_DELEGATEEXECUTE
	ASSOCSTR_SUPPORTED_URI_PROTOCOLS
	ASSOCSTR_PROGID
	ASSOCSTR_APPID
	ASSOCSTR_APPPUBLISHER
	ASSOCSTR_APPICONREFERENCE
	ASSOCSTR_MAX
)

func SafeAssocQueryString(
	flags int32,
	str int32,
	assoc string,
	extra string,
) (string, error) {
	assocPtr, err := windows.UTF16PtrFromString(assoc)
	if err != nil {
		return "", errors.Wrap(err, "error converting assoc to UTF16")
	}
	extraPtr, err := windows.UTF16PtrFromString(extra)
	if err != nil {
		return "", errors.Wrap(err, "error converting extra to UTF16")
	}
	var cch uint32
	if err := AssocQueryString(
		flags,
		str,
		assocPtr,
		extraPtr,
		nil,
		&cch,
	); err != nil {
		return "", errors.Wrap(err, "error pre-calling AssocQueryString")
	}
	buf := make([]uint16, cch+1)
	if err := AssocQueryString(
		flags,
		str,
		assocPtr,
		extraPtr,
		&buf[0],
		&cch,
	); err != nil {
		return "", errors.Wrap(err, "error calling AssocQueryString")
	}
	return windows.UTF16ToString(buf), nil
}

func AssocQueryString(
	flags int32,
	str int32,
	pszAssoc *uint16,
	pszExtra *uint16,
	pszOut *uint16,
	pcchOut *uint32,
) error {
	r0, _, e1 := procAssocQueryStringW.Call(
		uintptr(flags),
		uintptr(str),
		uintptr(unsafe.Pointer(pszAssoc)),
		uintptr(unsafe.Pointer(pszExtra)),
		uintptr(unsafe.Pointer(pszOut)),
		uintptr(unsafe.Pointer(pcchOut)),
	)
	if r0 != 0 && r0 != 1 {
		return e1
	}
	return nil
}
