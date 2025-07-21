package einomcphost

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试ServerConfig的GetTimeoutDuration方法
func TestServerConfig_GetTimeoutDuration(t *testing.T) {
	// 测试默认超时
	config := ServerConfig{}
	assert.Equal(t, time.Duration(DefaultMCPTimeoutSeconds)*time.Second, config.GetTimeoutDuration())

	// 测试自定义超时
	customTimeout := 60 * time.Second
	config.Timeout = customTimeout
	assert.Equal(t, customTimeout, config.GetTimeoutDuration())

	// 测试最小超时
	minTimeout := time.Duration(MinMCPTimeoutSeconds) * time.Second
	config.Timeout = minTimeout
	assert.Equal(t, minTimeout, config.GetTimeoutDuration())
}

// 测试validateSettings函数的各种情况
func TestValidateSettings_AdditionalCases(t *testing.T) {
	// 测试空的MCPServers
	emptySettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{},
	}
	err := validateSettings(emptySettings)
	assert.NoError(t, err)

	// 测试默认传输类型（空字符串）
	defaultTransportSettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"default_transport": {
				TransportType: "", // 默认为stdio
				Command:       "echo",
			},
		},
	}
	err = validateSettings(defaultTransportSettings)
	assert.NoError(t, err)

	// 测试刚好达到最小超时的配置
	minTimeoutSettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"min_timeout": {
				TransportType: "stdio",
				Command:       "echo",
				Timeout:       time.Duration(MinMCPTimeoutSeconds) * time.Second,
			},
		},
	}
	err = validateSettings(minTimeoutSettings)
	assert.NoError(t, err)
}

// 测试LoadSettingsFromString函数的边界情况
func TestLoadSettingsFromString_EdgeCases(t *testing.T) {
	// 测试空配置字符串
	settings, err := LoadSettingsFromString("")
	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Empty(t, settings.MCPServers)

	// 测试空的JSON对象
	emptyJSON := `{}`
	settings, err = LoadSettingsFromString(emptyJSON)
	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Empty(t, settings.MCPServers)

	// 测试只有mcpServers字段但为空的情况
	emptyServersJSON := `{"mcpServers":{}}`
	settings, err = LoadSettingsFromString(emptyServersJSON)
	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.Empty(t, settings.MCPServers)

	// 测试无效的配置（验证失败）
	invalidConfigJSON := `{
		"mcpServers": {
			"invalid_server": {
				"transportType": "invalid"
			}
		}
	}`
	_, err = LoadSettingsFromString(invalidConfigJSON)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid settings")
}

// TestServerConfigIsSSETransport tests the IsSSETransport method
func TestServerConfigIsSSETransport(t *testing.T) {
	tests := []struct {
		name          string
		transportType string
		expected      bool
	}{
		{
			name:          "SSE transport",
			transportType: TransportTypeSSE,
			expected:      true,
		},
		{
			name:          "stdio transport",
			transportType: TransportTypeStdio,
			expected:      false,
		},
		{
			name:          "empty transport",
			transportType: "",
			expected:      false,
		},
		{
			name:          "unknown transport",
			transportType: "unknown",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServerConfig{
				TransportType: tt.transportType,
			}
			assert.Equal(t, tt.expected, config.IsSSETransport())
		})
	}
}

// TestServerConfigIsStdioTransport tests the IsStdioTransport method
func TestServerConfigIsStdioTransport(t *testing.T) {
	tests := []struct {
		name          string
		transportType string
		expected      bool
	}{
		{
			name:          "stdio transport",
			transportType: TransportTypeStdio,
			expected:      true,
		},
		{
			name:          "empty transport (defaults to stdio)",
			transportType: "",
			expected:      true,
		},
		{
			name:          "SSE transport",
			transportType: TransportTypeSSE,
			expected:      false,
		},
		{
			name:          "unknown transport",
			transportType: "unknown",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServerConfig{
				TransportType: tt.transportType,
			}
			assert.Equal(t, tt.expected, config.IsStdioTransport())
		})
	}
}

// TestLoadSettingsFromStringWithEmptyData tests loading settings from empty string
func TestLoadSettingsFromStringWithEmptyData(t *testing.T) {
	settings, err := LoadSettingsFromString("")

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.NotNil(t, settings.MCPServers)
	assert.Equal(t, 0, len(settings.MCPServers))
}

// TestLoadSettingsFromStringWithWhitespace tests loading settings from whitespace string
func TestLoadSettingsFromStringWithWhitespace(t *testing.T) {
	settings, err := LoadSettingsFromString("   ")

	assert.NoError(t, err)
	assert.NotNil(t, settings)
	assert.NotNil(t, settings.MCPServers)
	assert.Equal(t, 0, len(settings.MCPServers))
}

// TestValidateSettingsWithNilMCPServers tests validation with nil MCPServers map
func TestValidateSettingsWithNilMCPServers(t *testing.T) {
	settings := &MCPSettings{
		MCPServers: nil,
	}

	err := validateSettings(settings)

	assert.NoError(t, err)
	assert.NotNil(t, settings.MCPServers)
	assert.Equal(t, 0, len(settings.MCPServers))
}

// TestValidateServerConfigWithWhitespace tests server config validation with whitespace
func TestValidateServerConfigWithWhitespace(t *testing.T) {
	tests := []struct {
		name        string
		config      *ServerConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "SSE with empty URL",
			config: &ServerConfig{
				TransportType: TransportTypeSSE,
				URL:           "",
			},
			expectError: true,
			errorMsg:    "URL is required for SSE transport",
		},
		{
			name: "SSE with whitespace URL",
			config: &ServerConfig{
				TransportType: TransportTypeSSE,
				URL:           "   ",
			},
			expectError: true,
			errorMsg:    "URL is required for SSE transport",
		},
		{
			name: "stdio with empty command",
			config: &ServerConfig{
				TransportType: TransportTypeStdio,
				Command:       "",
			},
			expectError: true,
			errorMsg:    "command is required for stdio transport",
		},
		{
			name: "stdio with whitespace command",
			config: &ServerConfig{
				TransportType: TransportTypeStdio,
				Command:       "   ",
			},
			expectError: true,
			errorMsg:    "command is required for stdio transport",
		},
		{
			name: "default transport with empty command",
			config: &ServerConfig{
				TransportType: "",
				Command:       "", // TransportType为空，先检查url，再检查command，都没有的话，返回unsupported transport type
			},
			expectError: true,
			errorMsg:    "unsupported transport type",
		},
		{
			name: "valid SSE config",
			config: &ServerConfig{
				TransportType: TransportTypeSSE,
				URL:           "http://localhost:8080",
			},
			expectError: false,
		},
		{
			name: "valid stdio config",
			config: &ServerConfig{
				TransportType: TransportTypeStdio,
				Command:       "python",
				Args:          []string{"-m", "server"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServerConfig("test-server", tt.config)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
