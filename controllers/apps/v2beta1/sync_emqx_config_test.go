package v2beta1

import (
	"fmt"
	"testing"

	"github.com/rory-z/go-hocon"
	"github.com/stretchr/testify/assert"
)

func TestMergeDefaultConfig(t *testing.T) {
	t.Run("case1", func(t *testing.T) {
		config := ""
		got := mergeDefaultConfig(config)
		hoconConfig, err := hocon.ParseString(got)
		assert.Nil(t, err)
		assert.Equal(t, "1883", hoconConfig.GetString("listeners.tcp.default.bind"))
		assert.Equal(t, "8883", hoconConfig.GetString("listeners.ssl.default.bind"))
		assert.Equal(t, "8083", hoconConfig.GetString("listeners.ws.default.bind"))
		assert.Equal(t, "8084", hoconConfig.GetString("listeners.wss.default.bind"))
	})

	t.Run("case2", func(t *testing.T) {
		config := ""
		config += fmt.Sprintln("listeners.tcp.default.bind = 31883")
		config += fmt.Sprintln("listeners.ssl.default.bind = 38883")
		config += fmt.Sprintln("listeners.ws.default.bind  = 38083")
		config += fmt.Sprintln("listeners.wss.default.bind = 38084")

		got := mergeDefaultConfig(config)
		hoconConfig, err := hocon.ParseString(got)
		assert.Nil(t, err)
		assert.Equal(t, "31883", hoconConfig.GetString("listeners.tcp.default.bind"))
		assert.Equal(t, "38883", hoconConfig.GetString("listeners.ssl.default.bind"))
		assert.Equal(t, "38083", hoconConfig.GetString("listeners.ws.default.bind"))
		assert.Equal(t, "38084", hoconConfig.GetString("listeners.wss.default.bind"))
	})
}
