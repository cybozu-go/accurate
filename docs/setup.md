# Deploying Accurate

1. (Optional) Prepare cert-manager

    Accurate depends on [cert-manager][] to issue TLS certificate for admission webhooks.
    If cert-manager is not installed on your cluster, install it as follows:

    ```console
    $ curl -fsLO https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
    $ kubectl apply -f cert-manager.yaml
    ```

2. Download the release tar ball

    Visit https://github.com/cybozu-go/accurate/releases/latest and download the source tar ball.

3. Unpack the tar ball and edit `config.yaml` in the top directory

    Read [Configurations](config.md) for details.

4. Download `kustomize`

    In the top directory of unpacked source code, run `make kustomize`.

    ```console
    $ make kustomize
    ```

5. Apply the manifest of Accurate

    ```console
    $ ./bin/kustomize build . | kubectl apply -f -
    ```

6. (Optional) Apply additional RBAC resources

    Read [Configurations](config.md) for details.

[cert-manager]: https://cert-manager.io/
