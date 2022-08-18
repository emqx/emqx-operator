package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	fsnotify "github.com/fsnotify/fsnotify"
)

type reloaderWatcher struct {
	watcher   *fsnotify.Watcher
	fileCheck map[string][]byte
}

func main() {
	var username, password string
	var licenseFilePath, pluginConfigDir string
	var port int
	flag.StringVar(&username, "u", "admin", "username")
	flag.StringVar(&password, "p", "public", "password")
	flag.IntVar(&port, "P", 8081, "port")
	flag.Parse()

	loadUrl := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", "localhost", port),
	}

	if os.Getenv("EMQX_PLUGINS__ETC_DIR") != "" {
		pluginConfigDir = os.Getenv("EMQX_PLUGINS__ETC_DIR")
	}

	if os.Getenv("EMQX_LICENSE__FILE") != "" {
		licenseFilePath = os.Getenv("EMQX_LICENSE__FILE")
	}

	// generate watched file list
	fileList := generateWatchedFileList(pluginConfigDir, licenseFilePath)

	r, err := newReloaderWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer r.close()
	r.watchFileList(fileList)

	go func() {
		for {
			select {
			case event, ok := <-r.watcher.Events:
				if !ok {
					return
				}
				if event.Op == fsnotify.Remove {
					updated, err := r.updateWatcher(event.Name)
					if err != nil {
						log.Fatal(err)
					}
					if updated {
						if event.Name == licenseFilePath {
							bytes, err := os.ReadFile(event.Name)
							if err != nil {
								log.Fatal(err)
							}
							/*
								curl -X POST --basic -u admin:public -d @<(jq -sR '{license: .}' < path/to/new.license)
								"http://localhost:8081/api/v4/license/upload"
							*/
							loadUrl.Path = "/api/v4/license/upload"
							license := map[string]string{"license": string(bytes)}
							body, err := json.Marshal(license)
							if err != nil {
								log.Fatal(err)
							}
							requestWebhook(loadUrl, "POST", body, username, password)
						} else {
							_, fileName := filepath.Split(event.Name)
							pluginName := strings.TrimSuffix(fileName, ".conf")
							/*
								curl -i --basic -u admin:public -X PUT
								"http://localhost:8081/api/v4/plugins/emqx_delayed_publish/reload"
							*/
							loadUrl.Path = fmt.Sprintf("/api/v4/plugins/%s/reload", pluginName)
							requestWebhook(loadUrl, "PUT", nil, username, password)
						}
					}
				}

			case err, ok := <-r.watcher.Errors:
				if !ok {
					return
				}
				log.Println("watcher error:", err)
			}
		}
	}()

	done := make(chan bool)
	<-done
}

func generateWatchedFileList(pluginConfigDir, licenseFilePath string) []string {
	var fileList []string
	if licenseFilePath != "" {
		fileList = append(fileList, licenseFilePath)
	}
	pluginConfigs, err := ioutil.ReadDir(pluginConfigDir)
	if err != nil {
		log.Printf("read dir err: %v", err)
		return fileList
	}
	for _, pluginConfig := range pluginConfigs {
		if shouldIgnoreFile(pluginConfig.Name()) {
			log.Printf("%s should not add to watcher", pluginConfig.Name())
			continue
		}
		filePath := filepath.Join(pluginConfigDir, pluginConfig.Name())
		fileList = append(fileList, filePath)
	}
	return fileList
}

func newReloaderWatcher() (*reloaderWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("NewWathcher err: %v\n", err)
		return nil, err
	}
	return &reloaderWatcher{
		fileCheck: make(map[string][]byte),
		watcher:   watcher,
	}, nil
}

func (r *reloaderWatcher) close() {
	r.watcher.Close()
}

func (r *reloaderWatcher) updateWatcher(filePath string) (bool, error) {
	// Since its actually a symlink
	// we are watching its not Write events
	// we need to react on Remove.
	// and we must re-register the file to be watched
	_ = r.watcher.Remove(filePath)
	err := r.watcher.Add(filePath)
	if err != nil {
		log.Printf("watcher add file err:%v", err)
		return false, err
	}
	if !bytes.Equal(r.fileCheck[filePath], getMD5(filePath)) {
		log.Println("file changed:", filePath)
		r.fileCheck[filePath] = getMD5(filePath)
		return true, nil
	}
	return false, nil
}

func (r *reloaderWatcher) watchFileList(fileList []string) {
	for _, filePath := range fileList {
		if err := r.watcher.Add(filePath); err != nil {
			log.Printf("add plugin config to watcher err: %v", err)
			continue
		}
		r.fileCheck[filePath] = getMD5(filePath)
	}
}

func shouldIgnoreFile(fileName string) bool {
	compile := regexp.MustCompile("^emqx.*conf$")
	if isBlackFile(fileName) || !compile.MatchString(fileName) {
		return true
	}
	return false
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

func requestWebhook(url url.URL, method string, bodys []byte, username, password string) {
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(bodys))
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

func isBlackFile(fileName string) bool {
	var blackReloadList = []string{
		"emqx_management.conf",
		"emqx_dashboard.conf",
	}
	for _, blackFile := range blackReloadList {
		if blackFile == fileName {
			return true
		}
	}
	return false
}
