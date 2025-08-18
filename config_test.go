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
				Transport: "", // 默认为stdio
				Command:   "echo",
			},
		},
	}
	err = validateSettings(defaultTransportSettings)
	assert.NoError(t, err)

	// 测试刚好达到最小超时的配置
	minTimeoutSettings := &MCPSettings{
		MCPServers: map[string]*ServerConfig{
			"min_timeout": {
				Transport: "stdio",
				Command:   "echo",
				Timeout:   time.Duration(MinMCPTimeoutSeconds) * time.Second,
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
				"transport": "invalid"
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
		name      string
		transport transport
		expected  bool
	}{
		{
			name:      "SSE transport",
			transport: transportSSE,
			expected:  true,
		},
		{
			name:      "stdio transport",
			transport: transportStdio,
			expected:  false,
		},
		{
			name:      "empty transport",
			transport: "",
			expected:  false,
		},
		{
			name:      "unknown transport",
			transport: "unknown",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServerConfig{
				Transport: tt.transport,
			}
			assert.Equal(t, tt.expected, config.IsSSETransport())
		})
	}
}

// TestServerConfigIsHTTPTransport tests the IsHTTPTransport method
func TestServerConfigIsHTTPTransport(t *testing.T) {
	tests := []struct {
		name      string
		transport transport
		expected  bool
	}{
		{
			name:      "HTTP transport",
			transport: transportHTTP1,
			expected:  true,
		},
		{
			name:      "streamable transport",
			transport: transportHTTPStreamable,
			expected:  true,
		},
		{
			name:      "SSE transport",
			transport: transportSSE,
			expected:  false,
		},
		{
			name:      "stdio transport",
			transport: transportStdio,
			expected:  false,
		},
		{
			name:      "empty transport",
			transport: "",
			expected:  false,
		},
		{
			name:      "unknown transport",
			transport: "unknown",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServerConfig{
				Transport: tt.transport,
			}
			assert.Equal(t, tt.expected, config.IsHTTPTransport())
		})
	}
}

// TestServerConfigIsStdioTransport tests the IsStdioTransport method
func TestServerConfigIsStdioTransport(t *testing.T) {
	tests := []struct {
		name      string
		transport transport
		expected  bool
	}{
		{
			name:      "stdio transport",
			transport: transportStdio,
			expected:  true,
		},
		{
			name:      "empty transport (defaults to stdio)",
			transport: "",
			expected:  true,
		},
		{
			name:      "SSE transport",
			transport: transportSSE,
			expected:  false,
		},
		{
			name:      "HTTP transport",
			transport: transportHTTP1,
			expected:  false,
		},
		{
			name:      "streamable transport",
			transport: transportHTTPStreamable,
			expected:  false,
		},
		{
			name:      "unknown transport",
			transport: "unknown",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &ServerConfig{
				Transport: tt.transport,
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
				Transport: transportSSE,
				URL:       "",
			},
			expectError: true,
			errorMsg:    "URL is required for SSE transport",
		},
		{
			name: "SSE with whitespace URL",
			config: &ServerConfig{
				Transport: transportSSE,
				URL:       "   ",
			},
			expectError: true,
			errorMsg:    "URL is required for SSE transport",
		},
		{
			name: "stdio with empty command",
			config: &ServerConfig{
				Transport: transportStdio,
				Command:   "",
			},
			expectError: true,
			errorMsg:    "command is required for stdio transport",
		},
		{
			name: "stdio with whitespace command",
			config: &ServerConfig{
				Transport: transportStdio,
				Command:   "   ",
			},
			expectError: true,
			errorMsg:    "command is required for stdio transport",
		},
		{
			name: "default transport with empty command",
			config: &ServerConfig{
				Transport: "",
				Command:   "", // transport为空，先检查url，再检查command，都没有的话，返回unsupported transport type
			},
			expectError: true,
			errorMsg:    "unsupported transport type",
		},
		{
			name: "valid SSE config",
			config: &ServerConfig{
				Transport: transportSSE,
				URL:       "http://localhost:8080",
			},
			expectError: false,
		},
		{
			name: "valid stdio config",
			config: &ServerConfig{
				Transport: transportStdio,
				Command:   "python",
				Args:      []string{"-m", "server"},
			},
			expectError: false,
		},
		{
			name: "valid HTTP transport config",
			config: &ServerConfig{
				Transport: transportHTTP1,
				URL:       "http://localhost:8080/mcp",
			},
			expectError: false,
		},
		{
			name: "valid streamable transport config",
			config: &ServerConfig{
				Transport: transportHTTPStreamable,
				URL:       "http://localhost:8080/mcp",
			},
			expectError: false,
		},
		{
			name: "HTTP transport with empty URL",
			config: &ServerConfig{
				Transport: transportHTTP1,
				URL:       "",
			},
			expectError: true,
			errorMsg:    "URL is required for",
		},
		{
			name: "streamable transport with empty URL",
			config: &ServerConfig{
				Transport: transportHTTPStreamable,
				URL:       "",
			},
			expectError: true,
			errorMsg:    "URL is required for",
		},
		{
			name: "HTTP transport with whitespace URL",
			config: &ServerConfig{
				Transport: transportHTTP1,
				URL:       "   ",
			},
			expectError: true,
			errorMsg:    "URL is required for",
		},
		{
			name: "streamable transport with whitespace URL",
			config: &ServerConfig{
				Transport: transportHTTPStreamable,
				URL:       "   ",
			},
			expectError: true,
			errorMsg:    "URL is required for",
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

// TestStreamableTransportAutoDetection 测试streamable transport的自动检测功能
func TestStreamableTransportAutoDetection(t *testing.T) {
	tests := []struct {
		name              string
		config            *ServerConfig
		expectedTransport transport
		expectError       bool
		errorMsg          string
	}{
		{
			name: "auto-detect SSE transport with /mcp suffix",
			config: &ServerConfig{
				Transport: "",
				URL:       "http://localhost:8080/mcp",
			},
			expectedTransport: transportSSE,
			expectError:       false,
		},
		{
			name: "auto-detect HTTP transport without /mcp suffix",
			config: &ServerConfig{
				Transport: "",
				URL:       "http://localhost:8080/api",
			},
			expectedTransport: transportHTTP1,
			expectError:       false,
		},
		{
			name: "auto-detect stdio transport with command",
			config: &ServerConfig{
				Transport: "",
				Command:   "python",
				Args:      []string{"-m", "server"},
			},
			expectedTransport: transportStdio,
			expectError:       false,
		},
		{
			name: "explicit streamable transport with URL",
			config: &ServerConfig{
				Transport: transportHTTPStreamable,
				URL:       "http://localhost:8080/stream",
			},
			expectedTransport: transportHTTPStreamable,
			expectError:       false,
		},
		{
			name: "explicit HTTP transport with URL",
			config: &ServerConfig{
				Transport: transportHTTP1,
				URL:       "http://localhost:8080/http",
			},
			expectedTransport: transportHTTP1,
			expectError:       false,
		},
		{
			name: "no URL and no command should fail",
			config: &ServerConfig{
				Transport: "",
			},
			expectedTransport: "",
			expectError:       true,
			errorMsg:          "unsupported transport type",
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
				assert.Equal(t, tt.expectedTransport, tt.config.Transport)
			}
		})
	}
}
