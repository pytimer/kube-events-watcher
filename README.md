# kube-events-watcher
This repository watch the event resource in the Kubernetes, which takes those events and push them to a specified sink.

## Usage

`./kube-events-watcher --kubeconfig kubeconfig.conf --sink=elasticsearch:http://elasticsearch:9200?maxRetries=5&index=events`

### Docker

`docker run -it --rm pytimer/kube-events-watcher:1.0.0 -h`

```sh
Usage of /kube-events-watcher:
      --kubeconfig string              absolute path to the kubeconfig file.
                                       Optional, if the kubeconfig empty, this controller is running in a kubernetes cluster.
      --log-flush-frequency duration   Maximum number of seconds between log flushes (default 5s)
      --resync-period duration         Watcher reflector resync period (default 1m0s)
      --sink *flags.Uri                Sink type to save the kubernetes events. e.g. --sink=elasticsearch:http://elasticsearch.com:9200
  -v, --v int                          log level for V logs
pflag: help requested
```
