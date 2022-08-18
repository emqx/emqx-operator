package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestIsBlackFile(t *testing.T) {
	testCases := []struct {
		fileName string
		expect   bool
	}{
		{
			fileName: "emqx_dashboard.conf",
			expect:   true,
		},
		{
			fileName: "emqx_exproto.conf",
			expect:   false,
		},
	}
	for _, test := range testCases {
		got := isBlackFile(test.fileName)
		assert.Equal(t, test.expect, got)
	}

}

func TestShouldIgnoreFile(t *testing.T) {
	testCases := []struct {
		blackList []string
		fileName  string
		expect    bool
	}{
		{
			fileName: "emqx_dashboard.conf",
			expect:   true,
		},
		{
			fileName: "emqx_exproto.conf",
			expect:   false,
		},
		{
			fileName: "emqx_dashboard.conf",
			expect:   true,
		},
	}
	for _, test := range testCases {
		got := shouldIgnoreFile(test.fileName)
		assert.Equal(t, test.expect, got)
	}
}

func TestGenerateFileCheck(t *testing.T) {
	testCases := []struct {
		licenseName      string
		pluginConfigName string
		expect           map[string][]byte
	}{
		{
			licenseName:      "",
			pluginConfigName: "",
			expect:           map[string][]byte{},
		},
		{
			licenseName:      "emqx.lic",
			pluginConfigName: "",
			expect:           map[string][]byte{},
		},
		{
			licenseName:      "",
			pluginConfigName: "emqx_exproto.conf",
			expect:           map[string][]byte{},
		},
		{
			licenseName:      "",
			pluginConfigName: "emqx_dashboard.conf",
			expect:           map[string][]byte{},
		},
	}
	for _, test := range testCases {
		r, _ := newReloaderWatcher()
		dir, _ := os.Getwd()
		fileList := []string{}
		if test.licenseName != "" {
			licenseFilePath := filepath.Join(dir, test.licenseName)
			f, err := os.Create(licenseFilePath)
			if err == nil {
				test.expect[licenseFilePath] = getMD5(licenseFilePath)
				fileList = append(fileList, licenseFilePath)
			}
			defer f.Close()
			defer os.Remove(licenseFilePath)
		}
		if test.pluginConfigName != "" {
			pluginConfigPath := filepath.Join(dir, test.pluginConfigName)
			f, err := os.Create(pluginConfigPath)
			if err == nil {
				test.expect[pluginConfigPath] = getMD5(pluginConfigPath)
				fileList = append(fileList, pluginConfigPath)
			}
			defer f.Close()
			defer os.Remove(pluginConfigPath)
		}
		r.watchFileList(fileList)
		assert.Equal(t, test.expect, r.fileCheck)
	}
}
