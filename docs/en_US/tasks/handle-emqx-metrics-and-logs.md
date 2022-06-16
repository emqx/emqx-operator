# Manage EMQX metrics and logs with telegraf

## Install telegraf-operator and deploy telegraf

[Refer to docs of telegraf-operator](https://github.com/influxdata/telegraf-operator)

## Configure `emqx_prometheus` plugin

[Refer to docs of install-emqx-prometheus-plugin](../tasks/install-emqx-prometheus-plugin.md)

## Configure class.yaml

```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: telegraf-operator-classes
     namespace: telegraf-operator
   stringData:
     ...
     # configuration about collecting metrics
     metrics: |+
       # please configure the ApiPort、EmqxApiUsername、EmqxApiPassword correctlly
       [[inputs.http]]
         urls = ["http://127.0.0.1:{{ApiPort}}/api/v4/emqx_prometheus"]
         method = "GET"
         timeout = "5s"
         username = "{{EmqxApiUsername}}"
         password = "{{EmqxApiPassword}}"
         data_format = "json"
       [inputs.http.tags]
         collection = "metric"        
     ...
     logs: |+
       [[inputs.tail]]
         files = ["/opt/emqx/log/emqx.log.[1-9]"]
         from_beginning = false
         max_undelivered_lines = 64
         character_encoding = "utf-8"
         data_format = "grok"
         grok_patterns = ['^%{TIMESTAMP_ISO8601:timestamp:ts-"2006-01-02T15:04:05.999999999-07:00"} \[%{LOGLEVEL:level}\] (?m)%{GREEDYDATA:messages}$']
       [inputs.tail.tags]
         collection = "log"
      ...
   ```

[Please refer to the document to configure outputs plugins you choosed](https://docs.influxdata.com/telegraf/v1.22/plugins/)