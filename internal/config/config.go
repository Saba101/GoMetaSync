package config

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)

type DBConfig struct {
    Name               string `yaml:"name"`
    DSN                string `yaml:"dsn"`

    Host               string `yaml:"host"`
    Port               int    `yaml:"port"`
    Username           string `yaml:"username"`
    Password           string `yaml:"password"`
    Database           string `yaml:"database"`
    SSL                bool   `yaml:"ssl"`
    RejectUnauthorized bool   `yaml:"rejectUnauthorized"`
}

type Config struct {
    Env       string     `yaml:"env"`
    Databases []DBConfig `yaml:"databases"`
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}

// BuildDSN creates a DSN from explicit host/password config if DSN is not provided
func (db DBConfig) BuildDSN() string {
    if db.DSN != "" {
        return db.DSN // user already provided full DSN
    }

    sslMode := "disable"
    if db.SSL {
        sslMode = "require"
        if !db.RejectUnauthorized {
            sslMode = "disable" // or "allow" if needed
        }
    }

    return fmt.Sprintf(
        "postgres://%s:%s@%s:%d/%s?sslmode=%s",
        db.Username,
        db.Password,
        db.Host,
        db.Port,
        db.Database,
        sslMode,
    )
}
