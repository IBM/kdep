

kdep
====

kdep stands for Kube Deploy and is a thin wrapper around [Helm](https://helm.sh/) that does two things:
  1) Automatically provisions secrets from [Vault](https://www.vaultproject.io/) during deployment
  2) Simplifies chart configuration for multiple environments

Table of contents
-----------------
 - [How it works](#how-it-works)
 - [Key features](#key-features)
 - [Installation](#installation)
 - [Usage](#usage)
 - [Overview & conventions](#overview--conventions)
 - [Values files](#values-files)
 - [Working with secrets](#working-with-secrets)
   - [Authentication](#authentication)
   - [Referring to secrets in Vault directly](#referring-to-secrets-in-vault-directly)
   - [Using config file templates with references to secrets in Vault](#using-config-file-templates-with-references-to-secrets-in-vault)
 - [Related work](#related-work)
 - [Questions & suggestions](#questions--suggestions)

How it works
------------
`kdep` is written in bash and is the command you call to install and upgrade your app (instead of `helm install/upgrade`).  

When called, it:
  1) Looks at the values files in your chart to see which secrets to fetch from Vault.
  2) Reads those secrets and saves them as Kubernetes secrets, optionally running them through a template.
  3) Configures your chart to point to these Kubernetes secrets on the fly.
  4) Deploys your chart via Helm.

Key features
------------
 - The secrets are never stored to disk.
 - The name of each generated Kubernetes secret includes a hash of the secret itself, meaning:
   - Deployments and rollbacks are immutable - a secret change is only propagated through a new deploy.
   - Pods are automatically restarted when the name of the secret they reference changes.
 - One command for both installs and upgrades.
 - Tiny footprint (200 loc bash) makes it easily customizable.

Installation
------------
Download a [release](https://github.com/IBM/kdep/releases) and unzip it into your PATH.  
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

Overview & conventions
----------------------
 - Logical structure
   - Microservices
     - Every microservice lives in its own git repository.
     - Closely related microservices are grouped together and thought of as "Apps".
   - "Apps" (groups of microservices)
     - Every "App" has its own git repo.
     - The "App" git repo consists of Helm chart(s) - one chart for each of the microservices that make up this "App".
     - Charts in this repo are named using lowercase letters, numbers, and dashes, something like api-order-history.
     - The chart's first word ("api" from above example) is the Kubernetes namespace that this chart is deployed to.
     - Generally, all of the charts in a particular "App" belong to one namespace, and this namespace is used only by this "App", though this is not a requirement.
     - Inside each chart are staggered [values files](#values-files) which define two things:
       - The configuration used for deployment to different clusters.
       - The secrets this chart needs from Vault.
 - Continuous Integration (CI)
   - A CI job (Jenkins/Travis/etc) is configured for each microservice repository.
   - The CI job generally runs tests, builds a docker image, tags it with the git commit hash, and pushes it to a docker registry.
   - The last step of the CI job is to create a Pull Request to the "App" repository, updating the image tag in the relevant chart.
 - Continuous Deployment (CD)
   - Structure
     - Pull Requests to an "App" repository can be deployed to test clusters and tested.
     - Merges into an "App" repository can be deployed to staging/production clusters.
     - An "App" repository can have multiple branches, for example, one for staging and one for production.
   - Approaches to deploying PRs and merges
     - Push-based approach
       - The idea is to push the changed charts into the clusters from a tool that runs outside the clusters.
       - Can be done using various tools like Jenkins/Travis/Spinnaker/etc.
     - Pull-based approach (cluster level)
       - The idea is something runs inside that cluster keeping it up to date.
       - Can be done using something like [Brigade](https://github.com/Azure/brigade)
     - Pull-based approach (application level)
       - The idea is every "App" in every cluster updates itself
       - Can be done via [quickcd](https://github.com/IBM/quickcd)
       - This is the approach used by the team behind `kdep`

Values files
------------
Part of most charts is a file named `values.yaml`. This file contains the default viles to be inserted into the chart's templates. This works well for deploying to one cluster but when deploying to multiple clusters, the values can be slightly different for each. A couple examples would be a different ingress domain, or different database connection parameters, etc. To store this information declaratively, we introduce the concept of staggered values files with the following naming convention: `[[region-]env-]values.yaml`.

Staggered values files work as follows: inside `values.yaml` we store all the most common configuration. Then in something like `prod-values.yaml` we store any differing values to be applied to clusters in the production environment. Then in something like `aus-prod-values.yaml` we store any values specific to production clusters running in the Australia region. Settings in a more specific values file will override the settings in the more general ones. These files are all optional and used only when needed - if all deployments of an app are the same, a single values file will suffice.

An example of how the staggered values files work:  
Let's say we want to launch the `api-order-history` chart into the production cluster in Australia. Let's assume this chart is already in our current working directory. The command for this scenario would be:  
`kdep api-order-history/aus-prod-values.yaml`  
When invoked with the above arguments, `kdep` will first read `values.yaml` and store the settings in memory. It would then read `prod-values.yaml` and merge the settings from it into the currently stored settings, overriding any differences. It would then read `aus-prod-values.yaml` and once again merge it into the currently stored settings, overriding any differences. In the end, the merged settings will be fed to the `helm install` command, which will provision the chart in the cluster we're set up to talk to. Internally, `kdep` uses kubectl and helm and therefore, talks to the cluster that these commands are configured to talk to in your current environment.

One thing to note for the example above is that if the file `aus-prod-values.yaml` does not exist, `kdep` will throw an error as it expects to be fed a real file path which we find to be more useful for development. When using kdep in a CI/CD context and deploying various charts, the `-i` flag is useful to ignore this error and allows `kdep` to be called in a standard way for all charts.

Working with secrets
--------------------
Before calling Helm to deploy a chart, kdep loads required secrets from Vault. It then creates those secrets in Kubernetes under the namespace of the chart. The name of each secret includes a hash of the secret itself, making for a unique name for each unique secret. After all the secrets are provisioned, kdep injects the generated secret names into the chart so that the Kubernetes secrets can be consumed by pods, as an env vars or volume mounts. Optionally, a configuration file template may be used into which kdep will inject secrets and then save the entire result as a Kubernetes secret that can be consumed by a pod through a volume mount.

### Authentication
Two authenticate with Vault there are two options. Option one is via the env var `VAULT_TOKEN`. If `VAULT_TOKEN` is not set, kdep uses option two, which gets two secrets from Kubernetes, `vault-role-id`, and `vault-secret-id`, which were once manually created. With this option, kdep will exchange the AppRole credentials (the two secrets) for a token and use the token to communicate with Vault from then on.

### Referring to secrets in Vault directly
This is done in a values file via the `kdep.secrets` key, for example:
```yaml
kdep:
  version: 1
  secrets:
    sec1: /generic/user/roman/myservice/sec1
    sec2: /generic/user/roman/myservice/sec2
    envSecret: /generic/user/roman/myservice/sec3
```
The above defines three secrets and specifies Vault paths for each. During deployment and before passing the values to Helm, kdep will fetch the referenced secrets, create them in the cluster, and turn the above into:
```yaml
kdep:
  version: 1
  secrets:
    sec1: name-of-dynamically-created-secret-in-cluster-sdf42
    sec2: name-of-dynamically-created-secret-in-cluster-4sdg6
    envSecret: name-of-dynamically-created-secret-in-cluster-f42zc
```
In other words, the secret paths are replaced by the generated secret names. This means that in the chart's template, this secret can be referenced like this: `{{ .Values.kdep.secrets.sec1 }}`.

There are two main ways for a pod to consume these secrets - by environment variable or by a filesystem mount.  
Here is an abridged  example of both:
```yaml
apiVersion: apps/v1beta2
kind: Deployment
metadata: ...
spec:
  template:
    ...
    spec:
      containers:
        - name: myservice
          image: "nginx:latest"
          env:
            - name: SECRET_FROM_ENV
              valueFrom:
                secretKeyRef:
                  name: {{ .Values.kdep.secrets.envSecret }}
                  key: value
          volumeMounts:
          - name: param1
            mountPath: "/tmp/sec1"
            subPath: value
          - name: param2
            mountPath: "/tmp/sec2"
            subPath: value
      volumes:
      - name: param1
        secret:
          secretName: {{ .Values.kdep.secrets.sec1 }}
      - name: param2
        secret:
          secretName: {{ .Values.kdep.secrets.sec2 }}
```

### Using config file templates with references to secrets in Vault
Sometimes, an application requires secrets in a configuration file. One way to accomplish this is to provide each secret to the container and then construct a special configuration file in the container's entrypoint script on startup. Kdep offers an alternative, perhaps simpler, solution. Instead of saving the configuration file template into the container, to be populated by the entrypoint script, save it to the chart folder. It will then be populated by kdep with secrets from Vault, saved as a secret in the cluster, and mounted in a pod as a volume.

Config file templates are specified in a values file via the `kdep.files` key:
```yaml
kdep:
  version: 1
  files:
    main_conf:
      template: main.conf.ctmpl
      secretPaths:
        hostname: /generic/user/roman/mysvc/hostname
        mysqlPassword: /generic/user/roman/mysvc/mysqlpassword
        kafkaBrokers: /generic/user/roman/mysvc/kafkabrokers
```
During deployment and before passing the values to Helm, kdep will run `main.conf.ctmpl` template through `consul-template`, save the output as a secret in the cluster, and turn the above into:
```yaml
kdep:
  version: 1
  files:
    main_conf: name-of-dynamically-created-secret-in-cluster-hje56
```

Let us disect the `kdep.files` specification above:
 - main_conf: this is a name used to refer to this config file secret in the chart's templates (see below)
 - template: this key specifies the filename of the configuration file template that will be filled with secrets. This file should be placed in the chart folder (same folder as Chart.yaml)
 - secretPaths: this maps Vault paths to friendly secret names. The Vault path can also be a "folder", for example if you intend to iterate through all items of a Vault "folder" inside the template.

The templating language used in the config file templates is that of consul-template and is very flexible.  
Here is an example of a config file template that uses the friendly names defined above:
```
service_hostname = {{ with secret "${hostname}" }}{{ .Data.value }}{{ end }}
mysql_password = {{ with secret "${mysqlPassword}" }}{{ .Data.value }}{{ end }}
kafka_servers = { {{ range secrets "${kafkaBrokers}" }}
  {{ with secret (printf "${kafkaBrokers}/%s" .) }}"{{ .Data.value }}";{{ end }}{{ end }}
}
```
During deployment, `kdep` will call `consul-template`, which will turn the above into the below, and store it as a secret in the cluster.
```
service_hostname = mainserver.local
mysql_password = 123456
kafka_servers = {
  "1.2.3.4";
  "5.6.7.8";
}
```
The details of the templating engine can be found here: https://github.com/hashicorp/consul-template#templating-language

Consuming config file secrets from a pod is the same as consuming regular secrets described above, here is an example:
```yaml
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  ...
spec:
  template:
    ...
    spec:
      containers:
        - name: myservice
          image: "nginx:latest"
          volumeMounts:
          - name: fileconf
            mountPath: "/tmp/main.conf"
            subPath: value
      volumes:
      - name: fileconf
        secret:
          secretName: {{ .Values.kdep.files.main_conf }}
```
The above configuration will make the populated version of `main.conf.ctmpl` available to the container at the path `/tmp/main.conf`.

Related work
------------
 - https://github.com/GoogleContainerTools/skaffold
 - https://github.com/roboll/kube-vault-controller

Questions & Suggestions
-----------------------
Please create an issue.
