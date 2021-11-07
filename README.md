# Prometheus Metric Dictionary
This project provides a convienent way to list and view details of available metrics in a prometheus setup.


## Quick example
- This short example assumes it is installed locally.
```
export PROMETHEUS_URL="http://demo.robustperception.io:9090"; prom_metrics_dictionary 
```
- Via docker 
```
docker run -e "PROMETHEUS_URL=http://demo.robustperception.io:9090" -p 8080:8080  tundeaoni/prometheus-metrics-dictionary
```
In both examples above the application would be available on the default port : `8080` i.e http://localhost:8080/

## Setup
##### Application Configuration
#

|  Name |  Description | Default Value  |
|---|---|---|
| PROMETHEUS_URL  |  URL to access prometheus |  required |
| REFRESH_INTERVAL  |  Time interval to update metrics details | 14400  (seconds)  |
| SERVE_PORT  |  Port to run application |  8080 |
| EXCLUDE_METRIC_LIST  |  Comma seperated list of metrics to exclude |  "" |

##### Installation
- Download a pre-compiled, [released version](https://github.com/tundeaoni/prometheus-metric-dictionary/releases).
- Extract the binary using unzip or tar.
- Move the binary into $PATH.