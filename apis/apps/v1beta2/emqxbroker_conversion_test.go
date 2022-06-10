package v1beta2_test

import (
	"testing"

	"github.com/emqx/emqx-operator/apis/apps/v1beta2"
	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		EmqxTemplate: v1beta2.EmqxBrokerTemplate{
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
		EmqxTemplate: v1beta3.EmqxBrokerTemplate{
			Image:      "emqx/emqx:4.4.1",
			EmqxConfig: map[string]string{"log.level": "debug"},
			Env: []corev1.EnvVar{
				{
					Name:  "EMQX_LOG__TO",
					Value: "both",
				},
			},
			Listener: v1beta3.Listener{
				Type: corev1.ServiceTypeNodePort,
				API: v1beta3.ListenerPort{
					Port:     int32(8081),
					NodePort: int32(8081),
				},
				MQTTS: v1beta3.ListenerPort{
					Port:     int32(8885),
					NodePort: int32(8885),
					Cert: v1beta3.CertConf{
						Data: v1beta3.CertData{
							CaCert:  []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURVVENDQWptZ0F3SUJBZ0lKQVBQWUNqVG14ZHQvTUEwR0NTcUdTSWIzRFFFQkN3VUFNRDh4Q3pBSkJnTlYKQkFZVEFrTk9NUkV3RHdZRFZRUUlEQWhvWVc1bmVtaHZkVEVNTUFvR0ExVUVDZ3dEUlUxUk1ROHdEUVlEVlFRRApEQVpTYjI5MFEwRXdIaGNOTWpBd05UQTRNRGd3TmpVeVdoY05NekF3TlRBMk1EZ3dOalV5V2pBL01Rc3dDUVlEClZRUUdFd0pEVGpFUk1BOEdBMVVFQ0F3SWFHRnVaM3BvYjNVeEREQUtCZ05WQkFvTUEwVk5VVEVQTUEwR0ExVUUKQXd3R1VtOXZkRU5CTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUF6Y2dWTGV4MQpFWjlPTjY0RVg4dit3Y1Nqek9acGlFT3NBT3VTWE9FTjN3YjhGS1V4Q2RzR3JzSllCN2E1Vk0vSm90MjVNb2QyCmp1UzNPQk1nNnI4NWsyVFdqZHhVb1VzK0hpVUIvcFAvQVJhYVc2Vm50cEFFb2twaWovcHJ6V01QZ0puQkYzVXIKTWp0YkxheUg5aEdtcFFySTVjMnZtSFEycmVSWm5TRmJZKzJiOFNYWiszbFpaZ3o5K0JhUVlXZFFXZmFVV0VIWgp1RGFOaVZpVk8wT1Q4RFJqQ3VpRHAzeVlEajNpTFdiVEEvZ0RMNlRmNVh1SHVFd2NPUVVyZCtoMGh5SXBoTzhECnRzcnNIWjE0ajRBV1lMazFDUEE2cHExSElVdkVsMnJBTngybFZVTnYrbnQ2NEsvTXIzUm5WUWQ5czhiSytUWFEKS0dIZDJMdi9QQUxZdXdJREFRQUJvMUF3VGpBZEJnTlZIUTRFRmdRVUdCbVcraUR6eGN0V0FXeG1oZ2RsRThQagpFYlF3SHdZRFZSMGpCQmd3Rm9BVUdCbVcraUR6eGN0V0FXeG1oZ2RsRThQakViUXdEQVlEVlIwVEJBVXdBd0VCCi96QU5CZ2txaGtpRzl3MEJBUXNGQUFPQ0FRRUFHYmhSVWpwSXJlZDRjRkFGSjdiYllEOWhLdS95eldQV2tNUmEKRXJsQ0tIbXVZc1lrKzVkMTZKUWhKYUZ5Nk1HWGZMZ28zS1YyaXRsMGQrT1dOSDBVOVVMWGNnbFR4eTYrbmpvNQpDRnFkVUJQd04xanhoem85eXRlRE1LRjQrQUhJeGJ2Q0FKYTE3cWN3VUtSNU1LTnZ2MDlDNnB2UURKTHppZDd5CkUyZGtnU3VnZ2lrM29hMDQyN0t2Y3RGZjh1aE9WOTRSdkVEeXF2VDUrcGdOWVoyWWZnYTlwRC9qanBvSEVVbG8KODhJR1U4L3dKQ3gzRHMyeWM4K29CZy95bnhHOGYvSG1DQzFFVDZFSEhvZTJqbG84RnBVL1NnR3RnaFMxWUwzMApJV3hOc1ByVVArWHNacEJKeS9tdk9oRTVRWG82WTM1ekRxcWo4dEk3QUdtQVd1MjJqZz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K"),
							TLSCert: []byte("LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURFekNDQWZ1Z0F3SUJBZ0lCQWpBTkJna3Foa2lHOXcwQkFRc0ZBREEvTVFzd0NRWURWUVFHRXdKRFRqRVIKTUE4R0ExVUVDQXdJYUdGdVozcG9iM1V4RERBS0JnTlZCQW9NQTBWTlVURVBNQTBHQTFVRUF3d0dVbTl2ZEVOQgpNQjRYRFRJd01EVXdPREE0TURjd05Wb1hEVE13TURVd05qQTRNRGN3TlZvd1B6RUxNQWtHQTFVRUJoTUNRMDR4CkVUQVBCZ05WQkFnTUNHaGhibWQ2YUc5MU1Rd3dDZ1lEVlFRS0RBTkZUVkV4RHpBTkJnTlZCQU1NQmxObGNuWmwKY2pDQ0FTSXdEUVlKS29aSWh2Y05BUUVCQlFBRGdnRVBBRENDQVFvQ2dnRUJBTE5lV1QzcEUrUUZmaVJKekttbgpBTVVyV28zSzJqL1RtMytYbmw2V0x6NjcvMHJjWXJKYmJLdlMzdXlSUC9zdFh5WEVLdzlDZXB5UTFWaUJWRmtXCkFveThxUUVPV0ZEc1pjLzVVemhYVW5iNkxYcjNxVGtGRWpObWhqKzd1enYvbGJCeGxVRzFObFl6U2VPQjYvUlQKOHpIL2xoT2VLaExuV1lQWGRYS3NhMUZMNmlqNFg4RGVETzFrWTdmdkFHbUJuL1RIaDF1VHBEaXpNNFltZUkrNwo0ZG1heUE1eFh2QVJ0ZTVoNFZ1NVNJemU3aUMwNTdOK3Z5bVRvTWsySmdrK1paRnB5WHJucSt5bzZSYUQzQU5jCmxyYzRGYmVVUVo1YTVzNVN4Z3M5YTBZM1dNRys3YzVWblZYY2JqQlJ6L2FxMk50T25RUWppa0tLUUE4R0YwODAKQlFrQ0F3RUFBYU1hTUJnd0NRWURWUjBUQkFJd0FEQUxCZ05WSFE4RUJBTUNCZUF3RFFZSktvWklodmNOQVFFTApCUUFEZ2dFQkFKZWZuTVpwYVJESFFTTlVJRUwzaXdHWEU5YzZQbUlzUVZFMnVzdHIrQ2FrQnAzVFo0bDBlbkx0CmlHTWZFVkZqdTY5Y080b3lva1d2K2hsNWVDTWtIQmYxNEt2NTF2ajQ0OGpvd1luRjF6bXpuN1NFem01VXpsc2EKc3FqdEFwcm5MeW9mNjlXdExVMWo1cllXQnVGWDg2eU9Ud1JBRk5qbTlmdmhBY3JFT05Cc1F0cWlwQldrTVJPcAppVVlNa1JxYktjUU1kd3hvditsSEJZS3E5emJXUm9xTFJPQW41NFNScWdRazZjMTVKZEVmZ09PalNoYnNPa0lIClVocWN3UmtRaWM3bjF6d0hWR1ZEZ05JWlZnbUoySWRJV0JsUEVDN29MclJyQkQvWDFpRUVYdEthYjZwNW8yMm4KS0I1bU4raVFhRStPZTJjcEdLWkppSlJkTStJcUREUT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo"),
							TLSKey:  []byte("LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb3dJQkFBS0NBUUVBczE1WlBla1Q1QVYrSkVuTXFhY0F4U3RhamNyYVA5T2JmNWVlWHBZdlBydi9TdHhpCnNsdHNxOUxlN0pFLyt5MWZKY1FyRDBKNm5KRFZXSUZVV1JZQ2pMeXBBUTVZVU94bHovbFRPRmRTZHZvdGV2ZXAKT1FVU00yYUdQN3U3Ty8rVnNIR1ZRYlUyVmpOSjQ0SHI5RlB6TWYrV0U1NHFFdWRaZzlkMWNxeHJVVXZxS1BoZgp3TjRNN1dSanQrOEFhWUdmOU1lSFc1T2tPTE16aGlaNGo3dmgyWnJJRG5GZThCRzE3bUhoVzdsSWpON3VJTFRuCnMzNi9LWk9neVRZbUNUNWxrV25KZXVlcjdLanBGb1BjQTF5V3R6Z1Z0NVJCbmxybXpsTEdDejFyUmpkWXdiN3QKemxXZFZkeHVNRkhQOXFyWTIwNmRCQ09LUW9wQUR3WVhUelFGQ1FJREFRQUJBb0lCQVFDdXZDYnI3UGQzbHZJLwpuN1ZGUUcrN3BIUmUxVkt3QXhEa3gydDhjWW9zN3kvUVdjbThQdHdxdHc1OEh6UFpHV1lyZ0dNQ1JwenprUlNGClY5ZzN3UDFTNVNjdTVDNmRCdTVZSUdjMTU3dHFOR1hCK1NwZFpkZEpRNE5jNnlHSFhZRVJsbFQwNGZmQkdjM04KV0cvb1lTLzFjU3RlaVNJcnNEeS85MUZ2R1JDaTdGUHhIM3dJZ0hzc1kvdHc2OXMxQ2Z2YXE1bHIyTlRGenhJRwp4Q3ZwSktFZFNmVmZTOUk3TFlpeW1WanN0M0lPUi93NzYvWkZZOWNSYThadG1RU1dXc20wVFVwUkMxamRjYmttClpvSnB0WVdsUCtnU3d4L2ZwTVlmdHJrSkZHT0poSEpIUWh3eFQ1WC9hakFJU2Vxamp3a1dTRUpMd25IUWQxMUMKWnkyKzI5bEJBb0dCQU5sRUFJSzRWeENxeVBYTktmb09PaTVkUzY0TmZ2eUg0QTF2MitLYUhXYzdscWFxUE40OQplemZOMm4zWCtLV3g0Y3ZpREQ5MTRZYzJKUTF2VkpqU2FIY2k3eWl2b2NEbzJPZlpEbWpCcXphTXAveStyWDFSCi9mM01taVRxTWE0NjhyamF4STlSUlp1N3ZEZ3BUUit6YTErT0JDZ016anZBbmc4ZEp1Ti81Z2psQW9HQkFOTlkKdVlQS3RlYXJCbWtxZHJTVjdlVFVlNDlOaHIwWG90TGFWQkgzN1RDVzBYdjl3ak8yeG1ibTVHYS9EQ3RQSXNCYgp5UGVZd1g5RmpvYXN1YWRVRDdoUnZiRnU2ZEJhMEhHTG1rWFJKWlRjRDdNRVgyTGh1NEJ1QzcyeURMTEZkMHIrCkVwOVdQN0Y1aUp5YWdZcUladHorNHVmN2dCdlVEZG12WHozc0dyMVZBb0dBZFhURDZlZUtlaUk2UGxoS0J6dEYKek9iM0VRT08wU3NMdjNmbm9kdTdaYUhiVWdMYW9UTVB1QjE3cjJqZ3JZTTdGS1FDQnhUTmRmR1ptbWZEamxMQgoweFo1d0w4aWJVMzBaWEw4elRsV1BFbFNUOXN0bzRCK0ZZVlZGL3ZjRzlzV2VVVWIybmNQY0ovUG8zVUFrdERHCmpZUVRUeXVOR3RTSkhwYWQvWU9aY3RrQ2dZQnRXUmFDN2JxM29mMHJKR0ZPaGRRVDlTd0l0Ti9scmZqOGh5SEEKT2pwcVRWNE5mUG1oc0F0dTZqOTZPWmFlUWMrRkh2Z1h3dDA2Y0U2UnQ0Ukc0dU5QUmx1VEZnTzdYWUZEZml0UAp2Q3Bwbm9JdzZTNUJCdkh3UFArdUloVVgyYnNpL2RtOHZ1OHRiK2dTdm80UGt3dEZoRXI2STlIZ2xCS21jbW9nCnE2d2FFUUtCZ0h5ZWNGQmVNNkxzMTFDZDY0dmJvcndKUEF1eElXN0hCQUZqL0JTOTlvZUc0VGpCeDRTejJkRmQKcnpVaWJKdDRuZG5ISXZDTjhKUWtqTkcxNGk5aEpsbitIM21Sc3M4ZmJaOXZRZHFHKzJ2T1dBRFlTenpzTkk1NQpSRlk3SmpsdUtjVmtwL3pDRGVVeFRVM082c1MrdjYvM1ZFMTFDb2I2T1lReDNsTjV3clozCi0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg"),
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

	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Type, v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.Type)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.API.Port, v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.Ports.API)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.API.NodePort, v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.NodePorts.API)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.MQTTS.Cert.StringData.CaCert, v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.Certificate.MQTTS.StringData.CaCert)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.MQTTS.Cert.StringData.TLSCert, v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.Certificate.MQTTS.StringData.TLSCert)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.MQTTS.Cert.StringData.TLSKey, v1beta2EmqxBroker.Spec.EmqxTemplate.Listener.Certificate.MQTTS.StringData.TLSKey)
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

	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Type, v1beta3EmqxBroker.Spec.EmqxTemplate.Listener.Type)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Ports.API, v1beta3EmqxBroker.Spec.EmqxTemplate.Listener.API.Port)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.NodePorts.API, v1beta3EmqxBroker.Spec.EmqxTemplate.Listener.API.NodePort)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Certificate.MQTTS.Data.CaCert, v1beta3EmqxBroker.Spec.EmqxTemplate.Listener.MQTTS.Cert.Data.CaCert)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Certificate.MQTTS.Data.TLSCert, v1beta3EmqxBroker.Spec.EmqxTemplate.Listener.MQTTS.Cert.Data.TLSCert)
	assert.Equal(t, emqx.Spec.EmqxTemplate.Listener.Certificate.MQTTS.Data.TLSKey, v1beta3EmqxBroker.Spec.EmqxTemplate.Listener.MQTTS.Cert.Data.TLSKey)
	assert.True(t, len(emqx.Spec.Env) == 2)
	assert.ElementsMatch(t, emqx.Spec.Env,
		[]corev1.EnvVar{
			{
				Name:  "EMQX_LOG__TO",
				Value: "both",
			},
			{
				Name:  "EMQX_LOG__LEVEL",
				Value: "debug",
			},
		})
}
