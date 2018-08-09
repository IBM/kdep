kdep
====

kdep stands for Kube Deploy and is a thin wrapper around [Helm](https://helm.sh/) that does two things:
  1) Automatically provisions secrets from [Vault](https://www.vaultproject.io/)
  2) Simplifies deployment configuration for multiple environments

How it works
------------
`kdep` is written in bash and is the command you call instead of `helm install/upgrade`.  

When called, it:
  1) Looks at the values files in your chart to see which secrets to fetch from Vault.
  2) Reads those secrets and saves them as Kubernetes secrets, optionally running them through a template.
  3) Configures your chart to point to these Kubernetes secrets on the fly.
  4) Deploys your chart via Helm.

Key benefits
------------
 - The secrets are never stored to disk.
 - The name of each generated Kubernetes secret includes a hash of the secret itself, meaning:
   - Deployments and rollbacks are immutable - a secret change is only propagated through a new deploy.
   - Pods are automatically restarted when the name of the secret they reference changes.
 - One command for both installs and upgrades.
 - Tiny footprint (200 loc bash) makes it easily customizable.

Installation
------------
Download a [release](https://github.com/IBM/kdep/releases) and unzip into your PATH.  
Required dependencies which you may already have:
 - [jq](https://stedolan.github.io/jq/download/)
 - [yq](https://github.com/mikefarah/yq/releases)
 - [Helm](https://github.com/helm/helm/releases)
 - [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
 - [Vault](https://www.vaultproject.io/downloads.html)
 - [consul-template](https://releases.hashicorp.com/consul-template/)

Usage
-----
```
kdep [-d] [-v] [-i] [-t temp-name] [-n namespace] path/to/chart/[[region-]env-]values.yaml

Flags:
  -d            Debug. Skips install/upgrade by adding '--dry-run --debug' to helm upgrade command 
  -v            Output values. Skips install/upgrade. This option makes the script create the secrets
                  in Kubernetes and then output the generated values file that would be passed to
                  helm upgrade.
  -i            Ignore if the passed in file doesn't exist. Useful mostly for cicd when we want to just
                  pass in a blanket command like 'kdep chart/useast-dev-values.yaml' not worrying
                  whether 'useast-dev-values.yaml' exists or only 'values.yaml' is present for example.
  -t temp-name  Temporary release name. Launch a chart meant to be deleted shortly after, useful for
                  testing. This flag specifies a custom release name. When - is specified, a release
                  name will be generated.
  -n namespace  Override the default namespace. The default namespace is inferred from the first word of
                  the chart name. When an argument is provided, the namespace will be added as a prefix
                  to the release name, which by default is the name of the chart.
```
