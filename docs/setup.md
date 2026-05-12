# Deploying Accurate

1. (Optional) Prepare cert-manager

    Accurate depends on [cert-manager][] to issue TLS certificate for admission webhooks.
    If cert-manager is not installed on your cluster, install it as follows:

    ```bash
    CERT_MANAGER_VERSION=v1.20.2
    CERT_MANAGER_GPG_KEY_URL=https://cert-manager.io/public-keys/cert-manager-keyring-2021-09-20-1020CF3C033D4F35BAE1C19E1226061C665DF13E.gpg
    curl -fsSL -o cert-manager-keyring.gpg "${CERT_MANAGER_GPG_KEY_URL}"
    helm pull oci://quay.io/jetstack/charts/cert-manager \
      --version "${CERT_MANAGER_VERSION}" \
      --verify --keyring cert-manager-keyring.gpg \
      --untar --untardir ./cert-manager-chart
    helm install cert-manager ./cert-manager-chart/cert-manager \
      --namespace cert-manager --create-namespace --set crds.enabled=true
    rm -rf cert-manager-keyring.gpg ./cert-manager-chart
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
