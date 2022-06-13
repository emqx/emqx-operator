package v1beta2_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var v1beta2EmqxBroker = v1beta2.EmqxBroker{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "broker",
		Namespace: "default",
		Labels: map[string]string{
			"cluster": "emqx",
		},
		Annotations: map[string]string{
			"cluster": "emqx",
		},
	},
	Spec: v1beta2.EmqxBrokerSpec{
		Image: "emqx/emqx:4.4.1",
		Labels: map[string]string{
			"foo": "bar",
		},
		Annotations: map[string]string{
			"foo": "bar",
		},
		Storage: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "foo",
				Value: "bar",
			},
			{
				Name:  "EMQX_LOG__TO",
				Value: "both",
			},
			{
				Name:  "EMQX_LOG__LEVEL",
				Value: "debug",
			},
			{
				Name:  "EMQX_MANAGEMENT_LISTENER__HTTP",
				Value: "8081",
			},
			{
				Name:  "EMQX_LISTENER__SSL__EXTERNAL",
				Value: "8885",
			},
		},
		EmqxTemplate: v1beta2.EmqxBrokerTemplate{
			ACL: []v1beta2.ACL{
				{
					Permission: "allow",
					Username:   "dashboard",
					Action:     "subscribe",
					Topics: v1beta2.Topics{
						Filter: []string{
							"$STS?#",
						},
					},
				},
				{
					Permission: "allow",
					IPAddress:  "127.0.0.1",
					Topics: v1beta2.Topics{
						Filter: []string{
							"$SYS/#",
							"#",
						},
					},
				},
				{
					Permission: "deny",
					Action:     "subscribe",
					Topics: v1beta2.Topics{
						Filter: []string{"$SYS/#"},
						Equal:  []string{"#"},
					},
				},
				{
					Permission: "allow",
				},
			},
			Listener: v1beta2.Listener{
				Type: corev1.ServiceTypeNodePort,
				Ports: v1beta2.Ports{
					API:   int32(8081),
					MQTTS: int32(8885),
				},
				NodePorts: v1beta2.Ports{
					API:   int32(8081),
					MQTTS: int32(8885),
				},
				Certificate: v1beta2.Certificate{
					MQTTS: v1beta2.CertificateConf{
						StringData: v1beta2.CertificateStringData{
							CaCert: `-----BEGIN CERTIFICATE-----
MIIDUTCCAjmgAwIBAgIJAPPYCjTmxdt/MA0GCSqGSIb3DQEBCwUAMD8xCzAJBgNV
BAYTAkNOMREwDwYDVQQIDAhoYW5nemhvdTEMMAoGA1UECgwDRU1RMQ8wDQYDVQQD
DAZSb290Q0EwHhcNMjAwNTA4MDgwNjUyWhcNMzAwNTA2MDgwNjUyWjA/MQswCQYD
VQQGEwJDTjERMA8GA1UECAwIaGFuZ3pob3UxDDAKBgNVBAoMA0VNUTEPMA0GA1UE
AwwGUm9vdENBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzcgVLex1
EZ9ON64EX8v+wcSjzOZpiEOsAOuSXOEN3wb8FKUxCdsGrsJYB7a5VM/Jot25Mod2
juS3OBMg6r85k2TWjdxUoUs+HiUB/pP/ARaaW6VntpAEokpij/przWMPgJnBF3Ur
MjtbLayH9hGmpQrI5c2vmHQ2reRZnSFbY+2b8SXZ+3lZZgz9+BaQYWdQWfaUWEHZ
uDaNiViVO0OT8DRjCuiDp3yYDj3iLWbTA/gDL6Tf5XuHuEwcOQUrd+h0hyIphO8D
tsrsHZ14j4AWYLk1CPA6pq1HIUvEl2rANx2lVUNv+nt64K/Mr3RnVQd9s8bK+TXQ
KGHd2Lv/PALYuwIDAQABo1AwTjAdBgNVHQ4EFgQUGBmW+iDzxctWAWxmhgdlE8Pj
EbQwHwYDVR0jBBgwFoAUGBmW+iDzxctWAWxmhgdlE8PjEbQwDAYDVR0TBAUwAwEB
/zANBgkqhkiG9w0BAQsFAAOCAQEAGbhRUjpIred4cFAFJ7bbYD9hKu/yzWPWkMRa
ErlCKHmuYsYk+5d16JQhJaFy6MGXfLgo3KV2itl0d+OWNH0U9ULXcglTxy6+njo5
CFqdUBPwN1jxhzo9yteDMKF4+AHIxbvCAJa17qcwUKR5MKNvv09C6pvQDJLzid7y
E2dkgSuggik3oa0427KvctFf8uhOV94RvEDyqvT5+pgNYZ2Yfga9pD/jjpoHEUlo
88IGU8/wJCx3Ds2yc8+oBg/ynxG8f/HmCC1ET6EHHoe2jlo8FpU/SgGtghS1YL30
IWxNsPrUP+XsZpBJy/mvOhE5QXo6Y35zDqqj8tI7AGmAWu22jg==
-----END CERTIFICATE-----`,
							TLSCert: `-----BEGIN CERTIFICATE-----
MIIDEzCCAfugAwIBAgIBAjANBgkqhkiG9w0BAQsFADA/MQswCQYDVQQGEwJDTjER
MA8GA1UECAwIaGFuZ3pob3UxDDAKBgNVBAoMA0VNUTEPMA0GA1UEAwwGUm9vdENB
MB4XDTIwMDUwODA4MDcwNVoXDTMwMDUwNjA4MDcwNVowPzELMAkGA1UEBhMCQ04x
ETAPBgNVBAgMCGhhbmd6aG91MQwwCgYDVQQKDANFTVExDzANBgNVBAMMBlNlcnZl
cjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALNeWT3pE+QFfiRJzKmn
AMUrWo3K2j/Tm3+Xnl6WLz67/0rcYrJbbKvS3uyRP/stXyXEKw9CepyQ1ViBVFkW
Aoy8qQEOWFDsZc/5UzhXUnb6LXr3qTkFEjNmhj+7uzv/lbBxlUG1NlYzSeOB6/RT
8zH/lhOeKhLnWYPXdXKsa1FL6ij4X8DeDO1kY7fvAGmBn/THh1uTpDizM4YmeI+7
4dmayA5xXvARte5h4Vu5SIze7iC057N+vymToMk2Jgk+ZZFpyXrnq+yo6RaD3ANc
lrc4FbeUQZ5a5s5Sxgs9a0Y3WMG+7c5VnVXcbjBRz/aq2NtOnQQjikKKQA8GF080
BQkCAwEAAaMaMBgwCQYDVR0TBAIwADALBgNVHQ8EBAMCBeAwDQYJKoZIhvcNAQEL
BQADggEBAJefnMZpaRDHQSNUIEL3iwGXE9c6PmIsQVE2ustr+CakBp3TZ4l0enLt
iGMfEVFju69cO4oyokWv+hl5eCMkHBf14Kv51vj448jowYnF1zmzn7SEzm5Uzlsa
sqjtAprnLyof69WtLU1j5rYWBuFX86yOTwRAFNjm9fvhAcrEONBsQtqipBWkMROp
iUYMkRqbKcQMdwxov+lHBYKq9zbWRoqLROAn54SRqgQk6c15JdEfgOOjShbsOkIH
UhqcwRkQic7n1zwHVGVDgNIZVgmJ2IdIWBlPEC7oLrRrBD/X1iEEXtKab6p5o22n
KB5mN+iQaE+Oe2cpGKZJiJRdM+IqDDQ=
-----END CERTIFICATE-----`,
							TLSKey: `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAs15ZPekT5AV+JEnMqacAxStajcraP9Obf5eeXpYvPrv/Stxi
sltsq9Le7JE/+y1fJcQrD0J6nJDVWIFUWRYCjLypAQ5YUOxlz/lTOFdSdvotevep
OQUSM2aGP7u7O/+VsHGVQbU2VjNJ44Hr9FPzMf+WE54qEudZg9d1cqxrUUvqKPhf
wN4M7WRjt+8AaYGf9MeHW5OkOLMzhiZ4j7vh2ZrIDnFe8BG17mHhW7lIjN7uILTn
s36/KZOgyTYmCT5lkWnJeuer7KjpFoPcA1yWtzgVt5RBnlrmzlLGCz1rRjdYwb7t
zlWdVdxuMFHP9qrY206dBCOKQopADwYXTzQFCQIDAQABAoIBAQCuvCbr7Pd3lvI/
n7VFQG+7pHRe1VKwAxDkx2t8cYos7y/QWcm8Ptwqtw58HzPZGWYrgGMCRpzzkRSF
V9g3wP1S5Scu5C6dBu5YIGc157tqNGXB+SpdZddJQ4Nc6yGHXYERllT04ffBGc3N
WG/oYS/1cSteiSIrsDy/91FvGRCi7FPxH3wIgHssY/tw69s1Cfvaq5lr2NTFzxIG
xCvpJKEdSfVfS9I7LYiymVjst3IOR/w76/ZFY9cRa8ZtmQSWWsm0TUpRC1jdcbkm
ZoJptYWlP+gSwx/fpMYftrkJFGOJhHJHQhwxT5X/ajAISeqjjwkWSEJLwnHQd11C
Zy2+29lBAoGBANlEAIK4VxCqyPXNKfoOOi5dS64NfvyH4A1v2+KaHWc7lqaqPN49
ezfN2n3X+KWx4cviDD914Yc2JQ1vVJjSaHci7yivocDo2OfZDmjBqzaMp/y+rX1R
/f3MmiTqMa468rjaxI9RRZu7vDgpTR+za1+OBCgMzjvAng8dJuN/5gjlAoGBANNY
uYPKtearBmkqdrSV7eTUe49Nhr0XotLaVBH37TCW0Xv9wjO2xmbm5Ga/DCtPIsBb
yPeYwX9FjoasuadUD7hRvbFu6dBa0HGLmkXRJZTcD7MEX2Lhu4BuC72yDLLFd0r+
Ep9WP7F5iJyagYqIZtz+4uf7gBvUDdmvXz3sGr1VAoGAdXTD6eeKeiI6PlhKBztF
zOb3EQOO0SsLv3fnodu7ZaHbUgLaoTMPuB17r2jgrYM7FKQCBxTNdfGZmmfDjlLB
0xZ5wL8ibU30ZXL8zTlWPElST9sto4B+FYVVF/vcG9sWeUUb2ncPcJ/Po3UAktDG
jYQTTyuNGtSJHpad/YOZctkCgYBtWRaC7bq3of0rJGFOhdQT9SwItN/lrfj8hyHA
OjpqTV4NfPmhsAtu6j96OZaeQc+FHvgXwt06cE6Rt4RG4uNPRluTFgO7XYFDfitP
vCppnoIw6S5BBvHwPP+uIhUX2bsi/dm8vu8tb+gSvo4PkwtFhEr6I9HglBKmcmog
q6waEQKBgHyecFBeM6Ls11Cd64vborwJPAuxIW7HBAFj/BS99oeG4TjBx4Sz2dFd
rzUibJt4ndnHIvCN8JQkjNG14i9hJln+H3mRss8fbZ9vQdqG+2vOWADYSzzsNI55
RFY7JjluKcVkp/zCDeUxTU3O6sS+v6/3VE11Cob6OYQx3lN5wrZ3
-----END RSA PRIVATE KEY-----`,
						},
					},
				},
			},
		},
	},
}
var v1beta3EmqxBroker = v1beta3.EmqxBroker{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "broker",
		Namespace: "default",
		Labels: map[string]string{
			"cluster": "emqx",
			"foo":     "bar",
		},
		Annotations: map[string]string{
			"cluster": "emqx",
			"foo":     "bar",
		},
	},
	Spec: v1beta3.EmqxBrokerSpec{
		Persistent: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				"ReadWriteOnce",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "foo",
				Value: "bar",
			},
		},
		EmqxTemplate: v1beta3.EmqxBrokerTemplate{
			Image: "emqx/emqx:4.4.1",
			EmqxConfig: map[string]string{
				"log.level":                "debug",
				"log.to":                   "both",
				"management_listener.http": "8081",
				"listener.ssl.external":    "8885",
			},
			ACL: []string{
				`{allow, {user, "dashboard"}, subscribe, ["$SYS/#"]}.`,
				`{allow, {ipaddr, "127.0.0.1"}, pubsub, ["$SYS/#", "#"]}.`,
				`{deny, all, subscribe, ["$SYS/#", {eq, "#"}]}.`,
				`{allow, all}.`,
			},
			ServiceTemplate: v1beta3.ServiceTemplate{
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Name:       "management-listener-http",
							Protocol:   corev1.ProtocolTCP,
							Port:       8081,
							TargetPort: intstr.FromInt(8081),
							NodePort:   8081,
						},
						{
							Name:       "listener-ssl-external",
							Protocol:   corev1.ProtocolTCP,
							Port:       8885,
							TargetPort: intstr.FromInt(8885),
							NodePort:   8885,
						},
					},
				},
			},
		},
	},
}

func TestConvertToBroker(t *testing.T) {
	emqx := &v1beta3.EmqxBroker{}
	err := v1beta2EmqxBroker.ConvertTo(emqx)

	assert.Nil(t, err)
	assert.Contains(t, emqx.Labels, "cluster")
	assert.Contains(t, emqx.Labels, "foo")

	assert.Contains(t, emqx.Annotations, "cluster")
	assert.Contains(t, emqx.Annotations, "foo")

	assert.Equal(t, emqx.Spec.Persistent, v1beta2EmqxBroker.Spec.Storage)

	aclList := &v1beta2.ACLList{
		Items: v1beta2EmqxBroker.Spec.EmqxTemplate.ACL,
	}
	assert.ElementsMatch(t, emqx.Spec.EmqxTemplate.ACL, aclList.Strings())

	assert.Subset(t, emqx.Spec.Env, []corev1.EnvVar{
		{
			Name:  "foo",
			Value: "bar",
		},
	})
	assert.Contains(t, emqx.Spec.EmqxTemplate.EmqxConfig["log.level"], "debug")
	assert.Contains(t, emqx.Spec.EmqxTemplate.EmqxConfig["log.to"], "both")

	assert.Equal(t, emqx.Spec.EmqxTemplate.ServiceTemplate.Spec.Type, v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.Type)
	assert.Equal(t, int32(8081), v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.Ports.API)
	assert.Equal(t, int32(8081), v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.NodePorts.API)
	assert.Equal(t, int32(8885), v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.Ports.MQTTS)
	assert.Equal(t, int32(8885), v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.NodePorts.MQTTS)
}

func TestConvertFromBroker(t *testing.T) {
	emqx := &v1beta2.EmqxBroker{}
	err := emqx.ConvertFrom(&v1beta3EmqxBroker)

	assert.Nil(t, err)
	assert.Contains(t, emqx.Labels, "cluster")
	assert.Contains(t, emqx.Labels, "foo")

	assert.Contains(t, emqx.Spec.Labels, "cluster")
	assert.Contains(t, emqx.Spec.Labels, "foo")

	assert.Contains(t, emqx.Annotations, "cluster")
	assert.Contains(t, emqx.Annotations, "foo")

	assert.Contains(t, emqx.Spec.Annotations, "cluster")
	assert.Contains(t, emqx.Spec.Annotations, "foo")

	assert.Equal(t, emqx.Spec.Storage, v1beta3EmqxBroker.Spec.Persistent)

	assert.Subset(t, emqx.Spec.Env,
		[]corev1.EnvVar{
			{
				Name:  "foo",
				Value: "bar",
			},
			{
				Name:  "EMQX_LOG__TO",
				Value: "both",
			},
			{
				Name:  "EMQX_LOG__LEVEL",
				Value: "debug",
			},
			{
				Name:  "EMQX_MANAGEMENT_LISTENER__HTTP",
				Value: "8081",
			},
			{
				Name:  "EMQX_LISTENER__SSL__EXTERNAL",
				Value: "8885",
			},
		})
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Type, v1beta3EmqxBroker.Spec.EmqxTemplate.ServiceTemplate.Spec.Type)

	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Ports.API, int32(8081))
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.NodePorts.API, int32(8081))

	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Ports.MQTTS, int32(8885))
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.NodePorts.MQTTS, int32(8885))
}
