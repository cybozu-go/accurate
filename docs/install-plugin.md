# Installing kubectl plugin

`kubectl-accurate` is a plugin for `kubectl` to make operations of Accurate easy.

It is strongly recommended to install `kubectl-accurate` though Accurate can be used without the plugin.

1. Set `OS` to the operating system name

    OS is one of `linux`, `windows`, or `darwin` (MacOS).

    If Go is available, `OS` can be set automatically as follows:

    ```console
    $ OS=$(go env GOOS)
    ```

1. Set `ARCH` to the operating system name

    OS is one of `amd64` or `arm64`.

    If Go is available, `ARCH` can be set automatically as follows:

    ```console
    $ OS=$(go env GOARCH)
    ```

2. Download the binary and put it in a directory of your `PATH`.

    The following is an example to install the plugin in `/usr/local/bin`.

    ```console
    $ sudo curl -o /usr/local/bin/kubectl-accurate -sLf \
      https://github.com/cybozu-go/accurate/releases/latest/download/kubectl-accurate-${OS}-${ARCH}
    $ sudo chmod a+x /usr/local/bin/kubectl-accurate
    ```

3. Check the installation

    Run `kubectl accurate -h` and see the output looks like:

    ```console
    $ kubectl accurate -h
    accurate is a subcommand of kubectl to manage Accurate features.

    Usage:
      accurate [command]

    Available Commands:
      completion  generate the autocompletion script for the specified shell
      help        Help about any command
      list        List namespace trees hierarchically
      namespace   namespace subcommand
      sub         sub-namespace command
      template    template subcommand
    ...
    ```
