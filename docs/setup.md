# Deploying Accurate

1. (Optional) Prepare cert-manager

    Accurate depends on [cert-manager][] to issue TLS certificate for admission webhooks.
    If cert-manager is not installed on your cluster, install it as follows:

    ```bash
    CERT_MANAGER_VERSION=v1.20.2
    CERT_MANAGER_SHA256=1ce11cae912adecc69e6bb623435fafc9ed21505f9efff98bd71d7b80f01db1f
    curl -fsSLO https://github.com/cert-manager/cert-manager/releases/download/${CERT_MANAGER_VERSION}/cert-manager.yaml
    echo "${CERT_MANAGER_SHA256}  cert-manager.yaml" | sha256sum --check
    kubectl apply -f cert-manager.yaml
    rm -f cert-manager.yaml
    ```

2. Setup Accurate Helm repository

   ```bash
   helm repo add accurate https://cybozu-go.github.io/accurate/
   helm repo update
   ```

3. Configuration Helm chart values

    Read [Configurations](config.md) for details.

4. Install the Accurate Helm chart

    ```bash
    helm install --create-namespace --namespace accurate accurate accurate/accurate -f values.yaml
    ```

[cert-manager]: https://cert-manager.io/
