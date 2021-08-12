package v1alpha1

import "k8s.io/apimachinery/pkg/api/resource"

//+kubebuilder:validation:Optional
type Log struct {
	//+kubebuilder:default:=both
	//+kubebuilder:validation:Enum:=off;file;console;both
	To LogOutput `json:"to,omitempty"`

	//+kubebuilder:default:=warning
	//+kubebuilder:validation:Enum:=debug;info;notice;warning;error;critical;alert;emergency
	Level LogLevel `json:"level,omitempty"`

	//+kubebuilder:default:="./log"
	Dir string `json:"dir,omitempty"`

	//+kubebuilder:default:=emqx.log
	File string `json:"file,omitempty"`

	//+kubebuilder:default:=-1
	CharsLimit int8 `json:"chars_limit,omitempty"`

	//+kubebuilder:default:=20
	MaxDepth int8 `json:"max_depth,omitempty"`

	Rotation Rotation `json:"rotation,omitempty"`

	//+kubebuilder:default:=True
	//+kubebuilder:validation:Enum:=True;False
	SingleLine bool `json:"single_line,omitempty"`

	//+kubebuilder:default:=text
	//+kubebuilder:validation:Enum:=text;json
	Formatter Formatter `json:"formatter,omitempty"`
}

type LogOutput string

const (
	LOG_OUTPUT_OFF     LogOutput = "off"
	LOG_OUTPUT_FILE    LogOutput = "file"
	LOG_OUTPUT_CONSOLE LogOutput = "console"
	LOG_OUTPUT_BOTH    LogOutput = "both"
)

type LogLevel string

const (
	LOG_LEVEL_DEBUG     LogOutput = "debug"
	LOG_LEVEL_INFO      LogOutput = "info"
	LOG_LEVEL_NOTICE    LogOutput = "notice"
	LOG_LEVEL_WARNING   LogOutput = "warning"
	LOG_LEVEL_ERROR     LogOutput = "error"
	LOG_LEVEL_CRITICAL  LogOutput = "critical"
	LOG_LEVEL_ALERT     LogOutput = "alert"
	LOG_LEVEL_EMERGENCY LogOutput = "emergency"
)

type Rotation struct {
	//+kubebuilder:default:="10MB"
	Size resource.Quantity `json:"size,omitempty"`

	//+kubebuilder:default:=5
	Count int8 `json:"count,omitempty"`
}

type Formatter string

const (
	FORMATTER_TEXT Formatter = "text"
	FORMATTER_JSON Formatter = "json"
)
