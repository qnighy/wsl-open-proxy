# wsl-open-proxy

wsl-open-proxy is a Windows application that can be registered
as default application handlers in WSL environments.

It launches the application registered as a default application
in the Windows side.

## Installation

### Installation using Go

You need to [have Go installed](https://go.dev/doc/install) in your WSL environment up front.

```console
$ go run github.com/qnighy/wsl-open-proxy/cmd/setup-wsl-open@latest
```

To configure the proxy for other filetypes than HTML, do for example:

```console
$ go run github.com/qnighy/wsl-open-proxy/cmd/setup-wsl-open@latest -t image
```

### Installing a prebuilt executable

To be filled later

## Development tips

When developing setup-wsl-open in Linux using VS Code, the following configuration might be useful:

```json
// .vscode/settings.json
{
    "go.toolsEnvVars": {
        "GOOS": "windows"
    }
}
```

## Releasing

```console
$ ./prebuild.sh
$ GOOS=... GOARCH=... go build ./cmd/setup-wsl-open
```

## License

Licensed under the MIT License.

## Related projects

[wsl-open](https://github.com/4U6U57/wsl-open) achieves the same purpose.
