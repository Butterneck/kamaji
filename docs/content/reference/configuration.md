## Configuration

Currently, **Kamaji** allows customization using CLI flags for the `manager` subcommand.

Available flags are the following:

| Flag | Usage | Default |
| ---- | ------ | --- |
| `--metrics-bind-address` | The address the metric endpoint binds to. | `:8080` |
| `--health-probe-bind-address` | The address the probe endpoint binds to. | `:8081` |
| `--leader-elect` | Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager. | `true` |
| `--tmp-directory` | Directory which will be used to work with temporary files. | `/tmp/kamaji` |
| `--kine-image` | Container image along with tag to use for the Kine sidecar container (used only if etcd-storage-type is set to one of kine strategies). | `rancher/kine:v0.9.2-amd64` |
| `--datastore` | The default DataStore that should be used by Kamaji to setup the required storage. | `etcd` |
| `--migrate-image` | Specify the container image to launch when a TenantControlPlane is migrated to a new datastore. | `migrate-image` |
| `--max-concurrent-tcp-reconciles` | Specify the number of workers for the Tenant Control Plane controller (beware of CPU consumption). | `1` |
| `--pod-namespace` | The Kubernetes Namespace on which the Operator is running in, required for the TenantControlPlane migration jobs. | `os.Getenv("POD_NAMESPACE")` |
| `--webhook-service-name` | The Kamaji webhook server Service name which is used to get validation webhooks, required for the TenantControlPlane migration jobs. | `kamaji-webhook-service` |
| `--serviceaccount-name` | The Kubernetes ServiceAccount used by the Operator, required for the TenantControlPlane migration jobs. | `os.Getenv("SERVICE_ACCOUNT")` |
| `--webhook-ca-path` | Path to the Manager webhook server CA, required for the TenantControlPlane migration jobs. | `/tmp/k8s-webhook-server/serving-certs/ca.crt` |
| `--zap-devel`  | Development Mode (encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode (encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error).  |  `true`  |
| `--zap-encoder`  | Zap log encoding, one of 'json' or 'console'  |  `console`  |
| `--zap-log-level`  |  Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity |  `info`  |
| `--zap-stacktrace-level`  | Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').  |  `info` |
| `--zap-time-encoding`  |  Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano') |  `epoch`  |
