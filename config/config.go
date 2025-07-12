package config

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	// "rate-limiter/types"
	"os"
	"reflect"
	"strings"
	"time"
)

// ConfigFile Path
const config_file_path = "application_config.json"

// LOGGER
var (
	// Look, I'm a java dev at heart and I like my logging levels
	InfoLogger  = log.New(os.Stdout, "[CONFIG] INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "[CONFIG] ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(os.Stdout, "[CONFIG] DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// Container for the full config
type Config struct {
	BackendURL        string           `json:"port"`
	JWTSecret         string           `json:"jwt_secret"`
	DefaultlimitCount int64            `json:"default_limit_count"` // Sensible, global default limit - unless over-ridden
	DefaultPeriod     time.Duration    `json:"default_period"`      // And a default time period
	MongoURL          string           `json:"mongo_url"`
	RedisURL          string           `json:"redis_url"`
	ServerConfig      HttpServerConfig `json:"server_config"`
}

// HTTP Listening Server config
type HttpServerConfig struct {
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// Load reads configuration from a file
func Load(filename string) (*Config, error) {

	var configFilePath string = config_file_path // Allow injecting an over-ride of the default config path for later use
	if strings.TrimSpace(filename) != "" {
		configFilePath = filename
	}

	var jsonData map[string]interface{}
	var err error

	jsonData, err = loadJSONConfig(configFilePath)
	if err != nil {
		InfoLogger.Printf("Unable to load JSON config file at %s", config_file_path)
	}

	// Acutally load things
	config := &Config{
		BackendURL:        getStringVal("backend_url", "http://localhost:8080", jsonData),
		JWTSecret:         getStringVal("jwt_secret", "your-secret-key", jsonData),
		DefaultlimitCount: getInt64("default_limit_count", 100, jsonData),
		DefaultPeriod:     getDuration("default_period", time.Hour, jsonData),
		MongoURL:          getStringVal("mongo_url", "mongodb://localhost:27017", jsonData),
		RedisURL:          getStringVal("redis_url", "redis://localhost:6379", jsonData),
		ServerConfig: HttpServerConfig{
			Port:         getInt("port", 8080, jsonData),
			ReadTimeout:  getDuration("read_timeout", 10*time.Second, jsonData),
			WriteTimeout: getDuration("write_timeout", 10*time.Second, jsonData),
			IdleTimeout:  getDuration("idle_timeout", 60*time.Second, jsonData),
		},
	}

	return config, config.Validate() // Return the config, and any errors when validating.
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errBuilder strings.Builder
	hasErrs := false
	if strings.TrimSpace(c.JWTSecret) == "" {
		errBuilder.WriteString("JWT secret cannot be empty\n")
		hasErrs = true
	}

	if len(c.JWTSecret) < 32 {
		errBuilder.WriteString("JWT is invalid\n")
		hasErrs = true
	}

	if c.ServerConfig.Port < 0 || c.ServerConfig.Port > 65535 {
		errBuilder.WriteString("Server port is invalid")
		hasErrs = true
	}

	if strings.TrimSpace(c.MongoURL) == "" {
		errBuilder.WriteString("MongoURL missing")
		hasErrs = true
	}

	if strings.TrimSpace(c.RedisURL) == "" {
		errBuilder.WriteString("RedisURL missing")
		hasErrs = true
	}

	if strings.TrimSpace(c.BackendURL) == "" {
		errBuilder.WriteString("BackendURL missing")
		hasErrs = true
	}
	// TODO
	// Add some actual content validation for the URLs
	// 	- Validate correct protocol for Mongo/Redis/HTTP
	//	- Validate path/host are legal
	// Actually validate the default timeouts

	if hasErrs {
		return fmt.Errorf("Missing required configuration, or Invalid config supplied:\n%s", errBuilder.String())
	}
	return nil // Yay no errors
}

func getConfigVal(key string, defaultVal interface{}, jsonData map[string]interface{}) interface{} {
	val := os.Getenv(key) // Env vars are primary
	if val != "" {
		return val
	}

	if jsonData != nil {
		if val, exists := jsonData[key]; exists {
			return val
		}
	}
	InfoLogger.Printf("Key %s not defined, load default %v", key, val)
	return defaultVal
}

func getStringVal(key string, defaultVal string, jsonData map[string]interface{}) string {
	result := getConfigVal(key, defaultVal, jsonData)
	if strVal, ok := result.(string); ok {
		return strVal
	}
	InfoLogger.Printf("Config %s loaded default value %s", key, defaultVal)
	return defaultVal
}

func getInt(key string, defaultVal int, jsonData map[string]interface{}) int {
	result := getConfigVal(key, defaultVal, jsonData)

	switch val := result.(type) {
	case string:
		if val == "" { // Empty string return default
			InfoLogger.Printf("Config %s is empty - loaded default value %v", key, defaultVal)
			return defaultVal
		}

		parsedVal, err := strconv.Atoi(val)
		if err != nil {
			ErrorLogger.Printf("invalid int value for %s: %s - Loaded default %v", key, val, defaultVal)
			return defaultVal
		}
		return parsedVal
	case float64:
		return int(val) // JSON numbers are float64, right?
	case int:
		return val
	default:
		InfoLogger.Printf("Unknown data type for key %s", key)
		return defaultVal
	}
}

func getInt64(key string, defaultVal int64, jsonData map[string]interface{}) int64 {
	result := getConfigVal(key, defaultVal, jsonData)

	switch val := result.(type) {
	case string:
		if val == "" {
			InfoLogger.Printf("Config %s is empty - loaded default value %v", key, defaultVal)
			return defaultVal
		}
		parsedVal, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			ErrorLogger.Printf("invalid int64 value for %s: %s - Loaded default %v", key, val, defaultVal)
			return defaultVal
		}
		return parsedVal
	case float64:
		return int64(val)
	case int64:
		return val
	default:
		InfoLogger.Printf("Unknown data type for key %s", key)
		return defaultVal
	}
}

func getDuration(key string, defaultValue time.Duration, jsonData map[string]interface{}) time.Duration {
	result := getConfigVal(key, defaultValue, jsonData)

	if str, ok := result.(string); ok {
		if str == "" {
			InfoLogger.Printf("Config %s is empty - loaded default value: %s", key, defaultValue)
			return defaultValue
		}
		parsed, err := time.ParseDuration(str)
		if err != nil {
			ErrorLogger.Printf("invalid Duration for %s: %s - Loaded default %v", key, str, defaultValue)
			return defaultValue
		}
		return parsed
	}
	InfoLogger.Printf("Config %s loaded default value %s", key, defaultValue)
	return defaultValue
}

// Load the JSON config file, IF IT EXISTS
func loadJSONConfig(filename string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err // File doesn't exist or can't read
	}

	var jsonConfig map[string]interface{}
	if err := json.Unmarshal(data, &jsonConfig); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return jsonConfig, nil
}

// Relfective loading...
// TODO/WIP
type FieldInfo struct {
	Name    string       // Go field name
	Type    reflect.Type // Go type
	JSONTag string       // JSON tag value
	GoType  string       // Human-readable type name
}

func GetStructFields(structValue interface{}) ([]FieldInfo, error) {
	var fields []FieldInfo
	structType := reflect.TypeOf(structValue)
	if structType.Kind() == reflect.Ptr { // Pointers we need the element type
		structType = structType.Elem()
	}

	// If it's not a struct, err out - this is *probably* fatal
	if structType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Invalid struct %s passed for reflective loading", structType.Kind())
	}

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip private fields (lowercase names)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		// Remove options like ",omitempty" with array slicing
		if comma := strings.Index(jsonTag, ","); comma != -1 {
			jsonTag = jsonTag[:comma]
		}

		fieldInfo := FieldInfo{
			Name:    field.Name,
			Type:    field.Type,
			JSONTag: jsonTag,
			GoType:  field.Type.String(),
		}
		fields = append(fields, fieldInfo)
	}
	return fields, nil
}
