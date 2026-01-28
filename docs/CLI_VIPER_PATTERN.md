# Viper Usage Pattern: Explicit Over Magic

## The Right Mental Model

```
Cobra = user intent
Viper = configuration resolution
Your code = final authority
```

**Viper is NOT a global singleton that your entire codebase imports.**

Instead: Use Viper only at the edges, then pass explicit structs.

---

## The Safe Pattern

### 1. Viper stays in config package

```go
// internal/config/config.go
package config

import (
    "github.com/spf13/viper"
)

// Config is the explicit configuration struct
// This is what the rest of the codebase sees
type Config struct {
    IAMMode      string
    Trace        bool
    PullOnStart  bool
    PolicyFile   string
    Ports        PortConfig
}

type PortConfig struct {
    IAM           int
    SecretManager int
    KMS           int
}

// Load reads from all sources and returns explicit Config
func Load() (*Config, error) {
    // Viper does the resolution (flags, env, file, defaults)
    initViper()
    
    // But we immediately marshal to explicit struct
    cfg := &Config{
        IAMMode:     viper.GetString("iam-mode"),
        Trace:       viper.GetBool("trace"),
        PullOnStart: viper.GetBool("pull-on-start"),
        PolicyFile:  viper.GetString("policy-file"),
        Ports: PortConfig{
            IAM:           viper.GetInt("port-iam"),
            SecretManager: viper.GetInt("port-secret-manager"),
            KMS:           viper.GetInt("port-kms"),
        },
    }
    
    // Validate
    if err := cfg.Validate(); err != nil {
        return nil, err
    }
    
    return cfg, nil
}

// Validate ensures config is sane
func (c *Config) Validate() error {
    if c.IAMMode != "off" && c.IAMMode != "permissive" && c.IAMMode != "strict" {
        return fmt.Errorf("invalid iam-mode: %s (must be off, permissive, or strict)", c.IAMMode)
    }
    
    if c.Ports.IAM < 1 || c.Ports.IAM > 65535 {
        return fmt.Errorf("invalid IAM port: %d", c.Ports.IAM)
    }
    
    // ... more validation
    
    return nil
}

// initViper sets up Viper (private, not exported)
func initViper() {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("$HOME/.gcp-emulator")
    viper.AddConfigPath(".")
    
    // Defaults
    viper.SetDefault("iam-mode", "permissive")
    viper.SetDefault("trace", false)
    viper.SetDefault("pull-on-start", false)
    viper.SetDefault("policy-file", "./policy.yaml")
    viper.SetDefault("port-iam", 8080)
    viper.SetDefault("port-secret-manager", 9090)
    viper.SetDefault("port-kms", 9091)
    
    // Environment variables
    viper.SetEnvPrefix("GCP_EMULATOR")
    viper.AutomaticEnv()
    
    // Read config file (ignore if not found)
    viper.ReadInConfig()
}

// Save writes current config to file
func Save(cfg *Config) error {
    viper.Set("iam-mode", cfg.IAMMode)
    viper.Set("trace", cfg.Trace)
    viper.Set("pull-on-start", cfg.PullOnStart)
    viper.Set("policy-file", cfg.PolicyFile)
    
    return viper.WriteConfig()
}

// Display shows current config (for `gcp-emulator config get`)
func Display() string {
    // Show exactly what the system computed
    cfg, _ := Load()
    
    return fmt.Sprintf(`Configuration:
  iam-mode:           %s
  trace:              %t
  pull-on-start:      %t
  policy-file:        %s
  
Ports:
  IAM:                %d
  Secret Manager:     %d
  KMS:                %d
  
Sources:
  Config file:        %s
  Environment:        GCP_EMULATOR_*
  Flags:              (per command)
`,
        cfg.IAMMode,
        cfg.Trace,
        cfg.PullOnStart,
        cfg.PolicyFile,
        cfg.Ports.IAM,
        cfg.Ports.SecretManager,
        cfg.Ports.KMS,
        viper.ConfigFileUsed(),
    )
}
```

---

### 2. Commands use explicit Config

```go
// internal/cli/start.go
package cli

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "github.com/blackwell-systems/gcp-iam-control-plane/internal/config"
    "github.com/blackwell-systems/gcp-iam-control-plane/internal/docker"
)

var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start the emulator stack",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Load config (Viper resolves behind the scenes)
        cfg, err := config.Load()
        if err != nil {
            return fmt.Errorf("invalid configuration: %w", err)
        }
        
        // 2. Pass explicit struct to business logic
        // NO viper.GetString() calls in business logic!
        return docker.Start(cfg)
    },
}

func init() {
    // Define flags
    startCmd.Flags().String("mode", "", "IAM mode (off|permissive|strict)")
    startCmd.Flags().Bool("pull", false, "Pull images before starting")
    
    // Bind to viper (so it participates in precedence)
    viper.BindPFlag("iam-mode", startCmd.Flags().Lookup("mode"))
    viper.BindPFlag("pull-on-start", startCmd.Flags().Lookup("pull"))
}
```

---

### 3. Business logic receives structs

```go
// internal/docker/compose.go
package docker

import (
    "github.com/blackwell-systems/gcp-iam-control-plane/internal/config"
)

// Start starts the docker compose stack
// NO viper imports here!
func Start(cfg *config.Config) error {
    // cfg is explicit, testable, serializable
    
    // Generate docker-compose environment
    env := []string{
        fmt.Sprintf("IAM_MODE=%s", cfg.IAMMode),
        fmt.Sprintf("IAM_PORT=%d", cfg.Ports.IAM),
        // ...
    }
    
    // Run docker-compose
    cmd := exec.Command("docker-compose", "up", "-d")
    cmd.Env = append(os.Environ(), env...)
    
    return cmd.Run()
}

// Stop stops the docker compose stack
func Stop() error {
    // Doesn't need config
    cmd := exec.Command("docker-compose", "down")
    return cmd.Run()
}

// Status returns health of services
func Status(cfg *config.Config) (*StackStatus, error) {
    // cfg tells us which ports to check
    
    status := &StackStatus{}
    
    // Check IAM health
    resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", cfg.Ports.IAM))
    if err != nil {
        status.IAM = ServiceDown
    } else if resp.StatusCode == 200 {
        status.IAM = ServiceUp
    }
    
    // ... check other services
    
    return status, nil
}
```

---

## Why This Pattern Works

### ✓ Explicit Config Struct

```go
// Good: config is explicit
func Start(cfg *config.Config) error {
    fmt.Printf("Starting with IAM mode: %s\n", cfg.IAMMode)
}

// Bad: config is invisible magic
func Start() error {
    mode := viper.GetString("iam-mode")  // Where did this come from?
}
```

### ✓ Testable

```go
func TestStart(t *testing.T) {
    cfg := &config.Config{
        IAMMode: "strict",
        Ports: config.PortConfig{
            IAM: 8080,
        },
    }
    
    // No Viper initialization needed!
    err := docker.Start(cfg)
    assert.NoError(t, err)
}
```

### ✓ Serializable

```go
// Can show users exactly what config was computed
func (c *Config) String() string {
    data, _ := yaml.Marshal(c)
    return string(data)
}
```

### ✓ Viper stays contained

```
internal/
├── config/        # ONLY place that imports viper
│   └── config.go
├── cli/           # Imports config, not viper
├── docker/        # Imports config, not viper
└── policy/        # Imports config, not viper
```

---

## The `gcp-emulator config get` Command

This is critical for user trust:

```bash
$ gcp-emulator config get

Configuration (resolved):
  iam-mode:           strict              (from: flag --mode)
  trace:              true                (from: env GCP_EMULATOR_TRACE)
  pull-on-start:      false               (from: default)
  policy-file:        ./policy.yaml       (from: config file)

Ports:
  IAM:                8080                (from: default)
  Secret Manager:     9090                (from: default)
  KMS:                9091                (from: default)

Sources used:
  Flags:              --mode=strict
  Environment:        GCP_EMULATOR_TRACE=true
  Config file:        /home/user/.gcp-emulator/config.yaml
  Defaults:           (built-in)

To change configuration:
  gcp-emulator config set iam-mode permissive
  export GCP_EMULATOR_TRACE=false
  gcp-emulator start --mode=strict
```

**Why this matters:**
- Users can see exactly what was resolved
- Debugging becomes trivial ("show me what the tool thinks")
- No surprise configuration

---

## What NOT To Do

### ❌ Viper imports everywhere

```go
// BAD: Docker package imports viper
package docker

import "github.com/spf13/viper"

func Start() error {
    mode := viper.GetString("iam-mode")  // Magic!
    // ...
}
```

**Why bad:**
- Can't tell where config comes from
- Can't test without global state
- Configuration becomes invisible

### ❌ No validation

```go
// BAD: Accept any string
IAMMode: viper.GetString("iam-mode")  // Could be "asdf"

// GOOD: Validate and error early
mode := viper.GetString("iam-mode")
if mode != "off" && mode != "permissive" && mode != "strict" {
    return fmt.Errorf("invalid iam-mode: %s", mode)
}
```

### ❌ Can't inspect final config

```go
// BAD: User has no way to see resolved config
// They run `gcp-emulator start` and it fails
// No way to debug "what did it think the config was?"

// GOOD: Always provide introspection
gcp-emulator config get
```

---

## Cobra + Viper Integration Points

### Root command initialization

```go
// cmd/gcp-emulator/main.go
func main() {
    // Initialize viper once at startup
    config.Init()
    
    // Execute cobra
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### Flag binding

```go
// Bind cobra flags to viper keys
func init() {
    startCmd.Flags().String("mode", "", "IAM mode")
    viper.BindPFlag("iam-mode", startCmd.Flags().Lookup("mode"))
}
```

### Load config in RunE

```go
RunE: func(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load()  // Viper resolves here
    if err != nil {
        return err
    }
    
    return docker.Start(cfg)  // Explicit config passed
}
```

---

## Strategic Alignment

Using Cobra + Viper (with this pattern) signals:

**✓ Production-grade tooling**
- Same stack as kubectl, docker, helm
- Professional configuration management
- Not a toy script

**✓ User-friendly**
- Flags, env vars, config file all work
- Clear precedence rules
- Introspectable with `config get`

**✓ Maintainable**
- Config is explicit structs
- Business logic has no magic
- Easy to test and reason about

**✓ Cloud-native positioning**
- Aligns with "local cloud control plane" narrative
- Serious infrastructure tooling
- Enterprise-ready patterns

---

## Summary

**Use Viper for:**
- Environment variables (`GCP_EMULATOR_*`)
- Single config file (`~/.gcp-emulator/config.yaml`)
- Flag binding (Cobra → Viper)
- Default values

**But always:**
- Load into explicit `Config` struct
- Pass structs, not viper.Get* calls
- Validate configuration
- Expose `config get` for introspection

**Never:**
- Import viper outside `internal/config/`
- Use `viper.Get*` in business logic
- Skip validation
- Hide configuration from users

This keeps Viper's benefits (precedence, env vars, config files) while avoiding the "invisible magic" trap.
