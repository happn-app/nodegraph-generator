---
title: Config
---

Config is loaded from `/etc/nodegraph-generator/config.yaml` (or from `CONFIG_PATH` if set)

```yaml
host: string
timeout: duration
prometheus_host: string
query_range: duration
query_step: duration
```

## host

The hostname of the API, should be `0.0.0.0:8080` (or another port)

## timeout

Duration after which the queries to prometheus will time out

## prometheus_host

The hostname of the prometheus server

## query_range

The range of time over which to query. Shorter means more accuracy, longer means better query performance

## query_step

The step of time between each data point in the range query.
