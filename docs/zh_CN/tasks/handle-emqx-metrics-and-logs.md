# 通过 Telegraf 采集处理 EMQX metrics 及 logs

## 安装 telegraf-operator 及 部署 telegraf 实例

[参考官方文档安装 telegraf-operator](https://github.com/influxdata/telegraf-operator)

## 配置 `emqx_prometheus` 插件

[具体步骤请参考](../tasks/install-emqx-prometheus-plugin.md)

## 自定义 class.yaml 配置

```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: telegraf-operator-classes
     namespace: telegraf-operator
   stringData:
     ...
     # Metric 采集配置
     metrics: |+
       # 请根据实际情况配置ApiPort、EmqxApiUsername、EmqxApiPassword
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

Metrics 及 Logs 的 outputs 插件可自行选择插件配置，[具体配置可参考文档](https://docs.influxdata.com/telegraf/v1.22/plugins/)