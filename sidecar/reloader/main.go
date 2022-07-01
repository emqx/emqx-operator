package main

import (
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	fsnotify "github.com/fsnotify/fsnotify"
)

const (
	method = "PUT"
	host   = "localhost"
)

var configFilePath = "/mounted/plugins/etc"

var fileCheck = make(map[string][]byte)

var configFileList = []string{
	"emqx_auth_http.conf",
	"emqx_auth_jwt.conf",
	"emqx_auth_ldap.conf",
	"emqx_auth_mnesia.conf",
	"emqx_auth_mongo.conf",
	"emqx_auth_mysql.conf",
	"emqx_auth_pgsql.conf",
	"emqx_auth_redis.conf",
	"emqx_backend_cassa.conf",
	"emqx_backend_dynamo.conf",
	"emqx_backend_influxdb.conf",
	"emqx_backend_mongo.conf",
	"emqx_backend_mysql.conf",
	"emqx_backend_opentsdb.conf",
	"emqx_backend_pgsql.conf",
	"emqx_backend_redis.conf",
	"emqx_backend_timescale.conf",
	"emqx_bridge_kafka.conf",
	"emqx_bridge_mqtt.conf",
	"emqx_bridge_pulsar.conf",
	"emqx_bridge_rabbit.conf",
	"emqx_bridge_rocket.conf",
	"emqx_coap.conf",
	"emqx_conf.conf",
	"emqx_dashboard.conf",
	"emqx_exhook.conf",
	"emqx_exproto.conf",
	"emqx_gbt32960.conf",
	"emqx_jt808.conf",
	"emqx_lua_hook.conf",
	"emqx_lwm2m.conf",
	// "emqx_management.conf",
	"emqx_modules.conf",
	"emqx_prometheus.conf",
	"emqx_psk_file.conf",
	"emqx_recon.conf",
	"emqx_retainer.conf",
	"emqx_rule_engine.conf",
	"emqx_sasl.conf",
	"emqx_schema_registry.conf",
	"emqx_sn.conf",
	"emqx_stomp.conf",
	"emqx_tcp.conf",
	"emqx_web_hook.conf",
}

func main() {
	var username string
	var password string
	var port int

	flag.StringVar(&username, "u", "admin", "username")
	flag.StringVar(&password, "p", "public", "password")
	flag.IntVar(&port, "P", 8081, "port")
	flag.Parse()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Remove {
					// Since its actually a symlink
					// we are watching its not Write events
					// we need to react on Remove.
					// and we must re-register the file to be watched
					_ = watcher.Remove(event.Name)
					err = watcher.Add(event.Name)
					if err != nil {
						log.Fatal(err)
					}
					if sha, ok := fileCheck[event.Name]; ok {
						if !bytes.Equal(sha, getMD5(event.Name)) {
							log.Println("file changed:", event.Name)
							fileCheck[event.Name] = getMD5(event.Name)

							_, fileName := filepath.Split(event.Name)
							name := strings.TrimSuffix(fileName, ".conf")

							url := url.URL{
								Scheme: "http",
								Host:   fmt.Sprintf("%s:%d", host, port),
								Path:   fmt.Sprintf("/api/v4/plugins/%s/reload", name),
							}

							requestWebhook(url, username, password)
						}
					}
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	for _, file := range configFileList {
		if os.Getenv("EMQX_PLUGINS__ETC_DIR") != "" {
			configFilePath = os.Getenv("EMQX_PLUGINS__ETC_DIR")
		}
		fullFile := filepath.Join(configFilePath, file)
		log.Printf("watching config file: %q", fullFile)

		err = watcher.Add(fullFile)
		if err != nil {
			log.Fatal(err)
		}

		fileCheck[fullFile] = getMD5(fullFile)
	}

	done := make(chan bool)
	<-done
}

func getMD5(file string) []byte {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return h.Sum(nil)
}

func requestWebhook(url url.URL, username, password string) {
	req, err := http.NewRequest(method, url.String(), nil)
	if err != nil {
		log.Println("http new request error:", err)
		return
	}
	req.SetBasicAuth(username, password)

	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("http client do error:", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("ioutil read error:", err)
		return
	}
	log.Println("http response url:", url.String())
	log.Println("status:", resp.Status)
	log.Println("body:", string(body))
}
