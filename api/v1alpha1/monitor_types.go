package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:validation:Optional
type Monitor struct {
	SysMon SysMon `json:"sysmon,omitempty"`

	OsMon SysMon `json:"os_mon,omitempty"`

	VmMon VmMon `json:"vm_mon,omitempty"`
}

type SysMon struct {

	//+kubebuilder:default:="0ms"
	LongGC metav1.Duration `json:"long_gc,omitempty"`

	//+kubebuilder:default:="240ms"
	LongSchedule metav1.Duration `json:"long_schedule,omitempty"`

	//+kubebuilder:default:="8MB"
	LargeHeap resource.Quantity `json:"large_heap,omitempty"`

	//+kubebuilder:default:=False
	BusyPort bool `json:"busy_port,omitempty"`

	//+kubebuilder:default:=True
	BusyDistPort bool `json:"busy_dist_port,omitempty"`
}

type OsMon struct {

	//+kubebuilder:default:="60s"
	CpuCheckInterval metav1.Duration `json:"cpu_check_interval,omitempty"`

	//+kubebuilder:default:="80%"
	//TODO
	CpuHighWatermark string `json:"cpu_high_watermark,omitempty"`

	//+kubebuilder:default:="60%"
	CpuLowWatermark string `json:"cpu_low_watermark,omitempty"`

	//+kubebuilder:default:="60s"
	MemCheckInterval metav1.Duration `json:"mem_check_interval,omitempty"`

	//+kubebuilder:default:="70%"
	SystemHighWatermark string `json:"sysmem_high_watermark,omitempty"`

	//+kubebuilder:default:="5%"
	ProcmemHighWatermark string `json:"procmem_high_watermark,omitempty"`
}

type VmMon struct {

	//+kubebuilder:default:="30s"
	CheckInterval metav1.Duration `json:"check_interval,omitempty"`

	//+kubebuilder:default:="80%"
	ProcessHighWatermark string `json:"process_high_watermark,omitempty"`

	//+kubebuilder:default:="60%"
	ProcessLowWatermark string `json:"process_low_watermark,omitempty"`
}
