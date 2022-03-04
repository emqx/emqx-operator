package v1beta2_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var v1beta2Enterprise = &v1beta2.EmqxEnterprise{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "emqx",
		Labels: map[string]string{
			"emqx": "cluster",
		},
		Annotations: map[string]string{
			"bar": "foo",
		},
	},
	Spec: v1beta2.EmqxEnterpriseSpec{
		Image: "emqx/emqx:4.3.11",
		Labels: map[string]string{
			"cluster": "emqx",
		},
		Annotations: map[string]string{
			"foo": "bar",
		},
		Storage: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
		},
		EmqxTemplate: v1beta2.EmqxEnterpriseTemplate{
			License: `-----BEGIN CERTIFICATE-----
MIIENzCCAx+gAwIBAgIDdMvVMA0GCSqGSIb3DQEBBQUAMIGDMQswCQYDVQQGEwJD
TjERMA8GA1UECAwIWmhlamlhbmcxETAPBgNVBAcMCEhhbmd6aG91MQwwCgYDVQQK
DANFTVExDDAKBgNVBAsMA0VNUTESMBAGA1UEAwwJKi5lbXF4LmlvMR4wHAYJKoZI
hvcNAQkBFg96aGFuZ3doQGVtcXguaW8wHhcNMjAwNjIwMDMwMjUyWhcNNDkwMTAx
MDMwMjUyWjBjMQswCQYDVQQGEwJDTjEZMBcGA1UECgwQRU1RIFggRXZhbHVhdGlv
bjEZMBcGA1UEAwwQRU1RIFggRXZhbHVhdGlvbjEeMBwGCSqGSIb3DQEJARYPY29u
dGFjdEBlbXF4LmlvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArw+3
2w9B7Rr3M7IOiMc7OD3Nzv2KUwtK6OSQ07Y7ikDJh0jynWcw6QamTiRWM2Ale8jr
0XAmKgwUSI42+f4w84nPpAH4k1L0zupaR10VYKIowZqXVEvSyV8G2N7091+6Jcon
DcaNBqZLRe1DiZXMJlhXnDgq14FPAxffKhCXiCgYtluLDDLKv+w9BaQGZVjxlFe5
cw32+z/xHU366npHBpafCbxBtWsNvchMVtLBqv9yPmrMqeBROyoJaI3nL78xDgpd
cRorqo+uQ1HWdcM6InEFET6pwkeuAF8/jJRlT12XGgZKKgFQTCkZi4hv7aywkGBE
JruPif/wlK0YuPJu6QIDAQABo4HSMIHPMBEGCSsGAQQBg5odAQQEDAIxMDCBlAYJ
KwYBBAGDmh0CBIGGDIGDZW1xeF9iYWNrZW5kX3JlZGlzLGVtcXhfYmFja2VuZF9t
eXNxbCxlbXF4X2JhY2tlbmRfcGdzcWwsZW1xeF9iYWNrZW5kX21vbmdvLGVtcXhf
YmFja2VuZF9jYXNzYSxlbXF4X2JyaWRnZV9rYWZrYSxlbXF4X2JyaWRnZV9yYWJi
aXQwEAYJKwYBBAGDmh0DBAMMATEwEQYJKwYBBAGDmh0EBAQMAjEwMA0GCSqGSIb3
DQEBBQUAA4IBAQDHUe6+P2U4jMD23u96vxCeQrhc/rXWvpmU5XB8Q/VGnJTmv3yU
EPyTFKtEZYVX29z16xoipUE6crlHhETOfezYsm9K0DxF3fNilOLRKkg9VEWcb5hj
iL3a2tdZ4sq+h/Z1elIXD71JJBAImjr6BljTIdUCfVtNvxlE8M0D/rKSn2jwzsjI
UrW88THMtlz9sb56kmM3JIOoIJoep6xNEajIBnoChSGjtBYFNFwzdwSTCodYkgPu
JifqxTKSuwAGSlqxJUwhjWG8ulzL3/pCAYEwlWmd2+nsfotQdiANdaPnez7o0z0s
EujOCZMbK8qNfSbyo50q5iIXhz2ZIGl+4hdp
-----END CERTIFICATE-----
`,
			Plugins: []v1beta3.Plugin{
				{
					Name:   "foo",
					Enable: true,
				},
				{
					Name:   "bar",
					Enable: false,
				},
			},
			Modules: []v1beta3.EmqxEnterpriseModule{
				{
					Name:   "emqx_mod_acl_internal",
					Enable: true,
				},
			},
			Listener: v1beta2.Listener{
				Ports: v1beta2.Ports{
					MQTTS: 8885,
				},
			},
		},
	},
}

var v1beta3Enterprise = &v1beta3.EmqxEnterprise{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "emqx",
		Namespace: "emqx",
		Labels: map[string]string{
			"cluster": "emqx",
			"emqx":    "cluster",
		},
		Annotations: map[string]string{
			"foo": "bar",
			"bar": "foo",
		},
	},
	Spec: v1beta3.EmqxEnterpriseSpec{
		Image: "emqx/emqx:4.3.11",
		Persistent: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
		},
		EmqxTemplate: v1beta3.EmqxEnterpriseTemplate{
			License: v1beta3.License{
				Data: []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUVOekNDQXgrZ0F3SUJBZ0lEZE12Vk1BMEdDU3FHU0liM0RRRUJCUVVBTUlHRE1Rc3dDUVlEVlFRR0V3SkQKVGpFUk1BOEdBMVVFQ0F3SVdtaGxhbWxoYm1jeEVUQVBCZ05WQkFjTUNFaGhibWQ2YUc5MU1Rd3dDZ1lEVlFRSwpEQU5GVFZFeEREQUtCZ05WQkFzTUEwVk5VVEVTTUJBR0ExVUVBd3dKS2k1bGJYRjRMbWx2TVI0d0hBWUpLb1pJCmh2Y05BUWtCRmc5NmFHRnVaM2RvUUdWdGNYZ3VhVzh3SGhjTk1qQXdOakl3TURNd01qVXlXaGNOTkRrd01UQXgKTURNd01qVXlXakJqTVFzd0NRWURWUVFHRXdKRFRqRVpNQmNHQTFVRUNnd1FSVTFSSUZnZ1JYWmhiSFZoZEdsdgpiakVaTUJjR0ExVUVBd3dRUlUxUklGZ2dSWFpoYkhWaGRHbHZiakVlTUJ3R0NTcUdTSWIzRFFFSkFSWVBZMjl1CmRHRmpkRUJsYlhGNExtbHZNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXJ3KzMKMnc5QjdScjNNN0lPaU1jN09EM056djJLVXd0SzZPU1EwN1k3aWtESmgwanluV2N3NlFhbVRpUldNMkFsZThqcgowWEFtS2d3VVNJNDIrZjR3ODRuUHBBSDRrMUwwenVwYVIxMFZZS0lvd1pxWFZFdlN5VjhHMk43MDkxKzZKY29uCkRjYU5CcVpMUmUxRGlaWE1KbGhYbkRncTE0RlBBeGZmS2hDWGlDZ1l0bHVMRERMS3YrdzlCYVFHWlZqeGxGZTUKY3czMit6L3hIVTM2Nm5wSEJwYWZDYnhCdFdzTnZjaE1WdExCcXY5eVBtck1xZUJST3lvSmFJM25MNzh4RGdwZApjUm9ycW8rdVExSFdkY002SW5FRkVUNnB3a2V1QUY4L2pKUmxUMTJYR2daS0tnRlFUQ2taaTRodjdheXdrR0JFCkpydVBpZi93bEswWXVQSnU2UUlEQVFBQm80SFNNSUhQTUJFR0NTc0dBUVFCZzVvZEFRUUVEQUl4TURDQmxBWUoKS3dZQkJBR0RtaDBDQklHR0RJR0RaVzF4ZUY5aVlXTnJaVzVrWDNKbFpHbHpMR1Z0Y1hoZlltRmphMlZ1WkY5dAplWE54YkN4bGJYRjRYMkpoWTJ0bGJtUmZjR2R6Y1d3c1pXMXhlRjlpWVdOclpXNWtYMjF2Ym1kdkxHVnRjWGhmClltRmphMlZ1WkY5allYTnpZU3hsYlhGNFgySnlhV1JuWlY5cllXWnJZU3hsYlhGNFgySnlhV1JuWlY5eVlXSmkKYVhRd0VBWUpLd1lCQkFHRG1oMERCQU1NQVRFd0VRWUpLd1lCQkFHRG1oMEVCQVFNQWpFd01BMEdDU3FHU0liMwpEUUVCQlFVQUE0SUJBUURIVWU2K1AyVTRqTUQyM3U5NnZ4Q2VRcmhjL3JYV3ZwbVU1WEI4US9WR25KVG12M3lVCkVQeVRGS3RFWllWWDI5ejE2eG9pcFVFNmNybEhoRVRPZmV6WXNtOUswRHhGM2ZOaWxPTFJLa2c5VkVXY2I1aGoKaUwzYTJ0ZFo0c3EraC9aMWVsSVhENzFKSkJBSW1qcjZCbGpUSWRVQ2ZWdE52eGxFOE0wRC9yS1NuMmp3enNqSQpVclc4OFRITXRsejlzYjU2a21NM0pJT29JSm9lcDZ4TkVhaklCbm9DaFNHanRCWUZORnd6ZHdTVENvZFlrZ1B1CkppZnF4VEtTdXdBR1NscXhKVXdoaldHOHVsekwzL3BDQVlFd2xXbWQyK25zZm90UWRpQU5kYVBuZXo3bzB6MHMKRXVqT0NaTWJLOHFOZlNieW81MHE1aUlYaHoyWklHbCs0aGRwCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"),
			},
			Plugins: []v1beta3.Plugin{
				{
					Name:   "foo",
					Enable: true,
				},
				{
					Name:   "bar",
					Enable: false,
				},
			},
			Modules: []v1beta3.EmqxEnterpriseModule{
				{
					Name:   "emqx_mod_acl_internal",
					Enable: true,
				},
			},
			Listener: v1beta3.Listener{
				MQTTS: v1beta3.Port{
					Port: int32(8885),
				},
			},
		},
	},
}

func TestConvertToEnterprise(t *testing.T) {
	emqx := &v1beta3.EmqxEnterprise{}
	err := v1beta2Enterprise.ConvertTo(emqx)

	v1beta3Enterprise.Spec.EmqxTemplate.License.Data = nil
	v1beta3Enterprise.Spec.EmqxTemplate.License.StringData = v1beta2Enterprise.Spec.EmqxTemplate.License

	assert.Nil(t, err)
	assert.Equal(t, emqx, v1beta3Enterprise)
}

func TestConvertFromEnterprise(t *testing.T) {
	emqx := &v1beta2.EmqxEnterprise{}
	err := emqx.ConvertFrom(v1beta3Enterprise)

	v1beta2Enterprise.Labels = v1beta3Enterprise.Labels
	v1beta2Enterprise.Annotations = v1beta3Enterprise.Annotations
	v1beta2Enterprise.Spec.Labels = v1beta3Enterprise.Labels
	v1beta2Enterprise.Spec.Annotations = v1beta3Enterprise.Annotations

	assert.Nil(t, err)
	assert.Equal(t, emqx, v1beta2Enterprise)
}
