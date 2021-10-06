# Installing kubectl plugin

`kubectl-accurate` is a plugin for `kubectl` to make operations of Accurate easy.

It is strongly recommended to install `kubectl-accurate` though Accurate can be used without the plugin.

## Installing using Krew

[Krew](https://krew.sigs.k8s.io/) is the plugin manager for kubectl command-line tool.

See the [documentation](https://krew.sigs.k8s.io/docs/user-guide/setup/install/) for how to install Krew.

```console
$ kubectl krew update
$ kubectl krew install accurate
```

## Installing manually

1. Set `OS` to the operating system name

    OS is one of `linux`, `windows`, or `darwin` (MacOS).

    If Go is available, `OS` can be set automatically as follows:

    ```console
    $ OS=$(go env GOOS)
    ```

2. Set `ARCH` to the operating system name

    ARCH is one of `amd64` or `arm64`.

    If Go is available, `ARCH` can be set automatically as follows:

    ```console
    $ ARCH=$(go env GOARCH)
    ```

3. Set `VERSION` to the accurate version

   See the accurate release page: https://github.com/cybozu-go/accurate/releases

   ```console
   $ VERSION=< The version you want to install >
   ```

4. Download the binary and put it in a directory of your `PATH`.

    The following is an example to install the plugin in `/usr/local/bin`.

    ```console
    $ curl -L -sS https://github.com/cybozu-go/accurate/releases/download/$(VERSION)/kubectl-accurate_$(VERSION)_$(OS)_$(ARCH).tar.gz \
      | tar xz -C /usr/local/bin kubectl-accurate
    ```

5. Check the installation

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
