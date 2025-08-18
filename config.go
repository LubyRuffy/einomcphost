// package einomcphost provides MCP (Model Context Protocol) server management functionality.
// It handles configuration loading, server connection management, and tool discovery
// for various MCP server implementations including stdio and SSE transports.
//
// The package supports:
//   - JSON-based configuration files
//   - Multiple transport types (stdio, SSE)
//   - Server validation and timeout management
//   - Graceful error handling for missing or invalid configurations
//
// Example usage:
//
//	settings, err := mcphost.LoadSettings("mcpservers.json")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	hub, err := mcphost.NewMCPHub(ctx, "mcpservers.json")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer hub.CloseServers()
package einomcphost

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/client"
)

// Timeout configuration constants define default and minimum timeout values
const (
	// DefaultMCPTimeoutSeconds is the default timeout for MCP operations in seconds
	DefaultMCPTimeoutSeconds = 30
	// MinMCPTimeoutSeconds is the minimum allowed timeout for MCP operations in seconds
	MinMCPTimeoutSeconds = 5
)

// Error message constants provide consistent error reporting
const (
	errMsgSettingsNil           = "settings cannot be nil"
	errMsgTimeoutTooSmall       = "server %s: timeout must be at least %d seconds"
	errMsgURLRequired           = "server %s: URL is required for SSE transport"
	errMsgCommandRequired       = "server %s: command is required for stdio transport"
	errMsgUnsupportedTransport  = "server %s: unsupported transport type: %s"
	errMsgFailedToParseSettings = "failed to parse settings: %w"
	errMsgInvalidSettings       = "invalid settings: %w"
	errMsgFailedToReadFile      = "failed to read settings file: %w"
)

// MCPSettings represents the main configuration structure for MCP servers.
// It contains a map of server configurations indexed by server name, allowing
// for multiple MCP servers to be configured and managed simultaneously.
//
// The configuration supports both enabled and disabled servers, with validation
// ensuring that all required fields are present for enabled servers.
type MCPSettings struct {
	MCPServers map[string]*ServerConfig `json:"mcpServers"`
}

// TransportType represents the type of transport used by an MCP server
type TransportType string

// Transport type constants define supported MCP transport mechanisms
const (
	TransportTypeSSE       TransportType = "sse"       // Server-Sent Events 最新版本已经弃用，使用 TransportTypeHTTP 代替
	TransportTypeHTTP      TransportType = "http"      // Streamable HTTP 流式 HTTP
	TransportTypeStdio     TransportType = "stdio"     // 标准输入输出
	TransportTypeInprocess TransportType = "inprocess" // 进程内 MCP 服务器，用于测试，实际上不在mcpServers中进行配置
)

// ServerConfig represents configuration for a single MCP server.
// It supports both SSE and stdio transport types with their respective settings.
// The configuration includes timeout management, auto-approval settings, and
// environment variable support for stdio-based servers.
//
// Transport-specific fields:
//   - SSE transport: Requires URL field
//   - Stdio transport: Requires Command field, optional Args and Env
//
// ref: https://github.com/mark3labs/mcphost/blob/4f2f61c6738417f74bc81c6910eebce632628569/internal/config/config.go
type ServerConfig struct {
	Transport   TransportType `json:"transport,omitempty" yaml:"transport,omitempty" mapstructure:"transport"`         // "sse" or "stdio" or "http" (defaults to "stdio")
	AutoApprove []string      `json:"autoApprove,omitempty" yaml:"auto_approve,omitempty" mapstructure:"auto_approve"` // List of auto-approved operations
	Disabled    bool          `json:"disabled,omitempty" yaml:"disabled,omitempty" mapstructure:"disabled"`            // Whether the server is disabled
	Timeout     time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`               // Operation timeout (defaults to 30s)

	// SSE specific configuration
	URL string `json:"url,omitempty" yaml:"url,omitempty" mapstructure:"url"` // Server URL for SSE transport

	// Stdio specific configuration
	Command string            `json:"command" yaml:"command" mapstructure:"command"`         // Command to execute for stdio transport
	Args    []string          `json:"args" yaml:"args" mapstructure:"args"`                  // Command arguments
	Env     map[string]string `json:"env,omitempty" yaml:"env,omitempty" mapstructure:"env"` // Environment variables

	// Inprocess specific configuration
	inProcessClient *client.Client `json:"inprocessClient,omitempty" yaml:"inprocessClient,omitempty" mapstructure:"inprocessClient"` // MCP client implementation to be used for this server
}

// GetTimeoutDuration returns the timeout duration for the server.
// If no timeout is configured, it returns the default timeout.
// This method ensures that all servers have a reasonable timeout value.
//
// Returns:
//   - time.Duration: The timeout duration, either configured or default
func (c *ServerConfig) GetTimeoutDuration() time.Duration {
	if c.Timeout == 0 {
		return time.Duration(DefaultMCPTimeoutSeconds) * time.Second
	}
	return c.Timeout
}

// IsSSETransport returns true if the server uses SSE transport.
// This method provides a convenient way to check the transport type
// without string comparison throughout the codebase.
//
// Returns:
//   - bool: true if the server uses SSE transport, false otherwise
func (c *ServerConfig) IsSSETransport() bool {
	return c.Transport == TransportTypeSSE || c.Transport == TransportTypeHTTP
}

// IsStdioTransport returns true if the server uses stdio transport.
// This includes both explicitly configured stdio transport and the default
// case where no transport type is specified (defaults to stdio).
//
// Returns:
//   - bool: true if the server uses stdio transport, false otherwise
func (c *ServerConfig) IsStdioTransport() bool {
	return c.Transport == TransportTypeStdio || c.Transport == ""
}

// LoadSettingsFromString loads MCP settings from a JSON configuration string.
// It parses the JSON and validates the resulting configuration to ensure
// all required fields are present and valid.
//
// The function handles empty strings gracefully by returning an empty but
// valid configuration, making it suitable for optional configuration scenarios.
//
// Parameters:
//   - data: JSON configuration string containing MCP server settings
//
// Returns:
//   - *MCPSettings: Parsed and validated settings ready for use
//   - error: Error if parsing or validation fails
//
// Example:
//
//	jsonConfig := `{"mcpServers": {"server1": {"command": "python", "args": ["-m", "server"]}}}`
//	settings, err := LoadSettingsFromString(jsonConfig)
//	if err != nil {
//		log.Fatal(err)
//	}
func LoadSettingsFromString(data string) (*MCPSettings, error) {
	dataStr := strings.TrimSpace(data)
	if dataStr == "" {
		return &MCPSettings{MCPServers: make(map[string]*ServerConfig)}, nil
	}

	var settings MCPSettings
	if err := json.Unmarshal([]byte(dataStr), &settings); err != nil {
		return nil, fmt.Errorf(errMsgFailedToParseSettings, err)
	}

	if err := validateSettings(&settings); err != nil {
		return nil, fmt.Errorf(errMsgInvalidSettings, err)
	}

	return &settings, nil
}

// LoadSettings loads MCP settings from a configuration file.
// It reads the file and delegates to LoadSettingsFromString for parsing.
// The function logs the file path being loaded for debugging purposes.
//
// Parameters:
//   - path: Path to the configuration file (JSON format expected)
//
// Returns:
//   - *MCPSettings: Parsed and validated settings from the file
//   - error: Error if file reading, parsing, or validation fails
//
// Example:
//
//	settings, err := LoadSettings("mcpservers.json")
//	if err != nil {
//		log.Fatal(err)
//	}
func LoadSettings(path string) (*MCPSettings, error) {
	log.Printf("Loading MCP settings from: %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(errMsgFailedToReadFile, err)
	}

	return LoadSettingsFromString(string(data))
}

// validateSettings validates the MCP settings configuration.
// It performs comprehensive validation including:
//   - Null safety checks
//   - Server-specific validation for each configured server
//   - Transport-specific requirement validation
//   - Timeout value validation
//
// The function initializes the MCPServers map if it's nil, ensuring
// the configuration is always in a valid state after validation.
//
// Parameters:
//   - settings: The settings to validate (will be modified if MCPServers is nil)
//
// Returns:
//   - error: Validation error if any configuration is invalid, nil otherwise
func validateSettings(settings *MCPSettings) error {
	if settings == nil {
		return fmt.Errorf(errMsgSettingsNil)
	}

	// Initialize MCPServers map if nil to ensure consistent state
	if settings.MCPServers == nil {
		settings.MCPServers = make(map[string]*ServerConfig)
	}

	// Validate each server configuration
	for name, server := range settings.MCPServers {
		if err := validateServerConfig(name, server); err != nil {
			return err
		}
	}

	return nil
}

// validateServerConfig validates a single server configuration.
// It performs transport-specific validation and ensures all required
// fields are present and valid for the specified transport type.
//
// Validation rules:
//   - Timeout must be at least MinMCPTimeoutSeconds if specified
//   - SSE transport requires a non-empty URL
//   - Stdio transport requires a non-empty Command
//   - Unknown transport types are rejected
//
// Parameters:
//   - name: Server name for error reporting
//   - server: Server configuration to validate
//
// Returns:
//   - error: Validation error if configuration is invalid, nil otherwise
func validateServerConfig(name string, server *ServerConfig) error {
	// Validate timeout if specified
	if server.Timeout > 0 && server.Timeout < time.Duration(MinMCPTimeoutSeconds)*time.Second {
		return fmt.Errorf(errMsgTimeoutTooSmall, name, MinMCPTimeoutSeconds)
	}

	// Validate transport-specific requirements
	switch server.Transport {
	case TransportTypeSSE, TransportTypeHTTP:
		if strings.TrimSpace(strings.TrimSpace(server.URL)) == "" {
			return fmt.Errorf(errMsgURLRequired, name)
		}
	case TransportTypeStdio: // Empty string defaults to stdio
		if strings.TrimSpace(strings.TrimSpace(server.Command)) == "" {
			return fmt.Errorf(errMsgCommandRequired, name)
		}
	case "":
		if strings.TrimSpace(strings.TrimSpace(server.URL)) != "" {
			// 简单的认为以 /mcp 结尾的 URL 是 SSE 传输，否则是 HTTP 传输
			if strings.HasSuffix(strings.ToLower(server.URL), "/mcp") {
				server.Transport = TransportTypeSSE
			} else {
				server.Transport = TransportTypeHTTP
			}
		} else if strings.TrimSpace(strings.TrimSpace(server.Command)) != "" {
			server.Transport = TransportTypeStdio
		} else {
			return fmt.Errorf(errMsgUnsupportedTransport, name, server.Transport)
		}
	default:
		return fmt.Errorf(errMsgUnsupportedTransport, name, server.Transport)
	}

	return nil
}
