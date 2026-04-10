# accurate-controller

`accurate-controller` is a Kubernetes controller to manage sub-namespaces and
to propagate resources from parents to their children namespaces.

## Configuration file

`accurate-controller` reads a configuration file on startup.
The default location is `/etc/accurate/config.yaml`.
The location can be changed with `--config-file` flag.

The configuration file should be a JSON or YAML file having the following keys:

| Key                          | Type       | Description                                                       |
| ---------------------------- | ---------- | ----------------------------------------------------------------- |
| `labelKeys`                  | `[]string` | Keys of namespace labels to be propagated.                        |
| `annotationKeys`             | `[]string` | Keys of namespace annotations to be propagated.                   |
| `subNamespaceLabelKeys`      | `[]string` | Keys of SubNamespace labels to be propagated.                     |
| `subNamespaceAnnotationKeys` | `[]string` | Keys of SubNamespace annotations to be propagated.                |
| `watches`                    | `[]object` | GroupVersionKind of namespace-scoped objects to be propagated.    |

Example:

```yaml
labelKeys:
- team

annotationKeys:
- foo.bar/baz

subNamespaceLabelKeys:
- app

subNamespaceAnnotationKeys:
- foo.bar/baz

watches:
- group: rbac.authorization.k8s.io
  version: v1
  kind: Role
- group: rbac.authorization.k8s.io
  version: v1
  kind: RoleBinding
- version: v1
  kind: Secret
```

## Environment variables

| Name            | Required | Description                                                |
| --------------- | -------- | ---------------------------------------------------------- |
| `POD_NAMESPACE` | Yes      | The namespace name where `accurate-controller` is running. |

## Command-line flags

```txt
Flags:
      --add_dir_header                     If true, adds the file directory to the header of the log messages
      --alsologtostderr                    log to standard error as well as files (no effect when -logtostderr=true)
      --apiserver-qps-throttle int         Maximum client-side QPS to the API server. Values greater than 0 enable throttling.
      --cert-dir string                    webhook certificate directory
      --config-file string                 Configuration file path (default "/etc/accurate/config.yaml")
      --feature-gates mapStringBool        A set of key=value pairs that describe feature gates for alpha/experimental features. Options are:
                                           AllAlpha=true|false (ALPHA - default=false)
                                           AllBeta=true|false (BETA - default=false)
                                           DisablePropagateGenerated=true|false (BETA - default=true)
      --health-probe-addr string           Listen address for health probes (default ":8081")
  -h, --help                               help for accurate-controller
      --leader-election-id string          ID for leader election by controller-runtime (default "accurate")
      --log_backtrace_at traceLocation     when logging hits line file:N, emit a stack trace (default :0)
      --log_dir string                     If non-empty, write log files in this directory (no effect when -logtostderr=true)
      --log_file string                    If non-empty, use this log file (no effect when -logtostderr=true)
      --log_file_max_size uint             Defines the maximum size a log file can grow to (no effect when -logtostderr=true). Unit is megabytes. If the value is 0, the maximum file size is unlimited. (default 1800)
      --logtostderr                        log to standard error instead of files (default true)
      --metrics-addr string                The address the metric endpoint binds to (default ":8080")
      --one_output                         If true, only write logs to their native severity level (vs also writing to each lower severity level; no effect when -logtostderr=true)
      --skip_headers                       If true, avoid header prefixes in the log messages
      --skip_log_headers                   If true, avoid headers when opening log files (no effect when -logtostderr=true)
      --stderrthreshold severity           logs at or above this threshold go to stderr when writing to files and stderr (no effect when -logtostderr=true or -alsologtostderr=true) (default 2)
  -v, --v Level                            number for the log level verbosity
      --version                            version for accurate-controller
      --vmodule moduleSpec                 comma-separated list of pattern=N settings for file-filtered logging
      --webhook-addr string                Listen address for the webhook endpoint (default ":9443")
      --webhook-allow-cascading-deletion   Set to true to allow cascading deletion of namespaces (namespaces with children)
      --zap-devel                          Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
      --zap-encoder encoder                Zap log encoding (one of 'json' or 'console')
      --zap-log-level level                Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', 'panic' or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
      --zap-stacktrace-level level         Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
      --zap-time-encoding time-encoding    Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.
```
