# Deploying Accurate

1. (Optional) Prepare cert-manager

    Accurate depends on [cert-manager][] to issue TLS certificate for admission webhooks.
    If cert-manager is not installed on your cluster, install it as follows:

    ```bash
    curl -fsLO https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
    kubectl apply -f cert-manager.yaml
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
