package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

var Conf *Config

type Config struct {
	DataBase *DataBase `yaml:"database"`
	Server   *Server   `yaml:"server"`
	SMTP     *SMTP     `yaml:"smtp"`
}

type DataBase struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Dbname   string `yaml:"db"`
}

type Server struct {
	IP   string `yaml:"ip"`
	Pass string `yaml:"pass"`
	Salt string `yaml:"salt"`
	GRPC struct {
		Port string `yaml:"port"`
	} `yaml:"grpc"`
	Web struct {
		Port   string `yaml:"port"`
		Prefix string `yaml:"prefix"`
	} `yaml:"web"`
	Static struct {
		FilePath string `yaml:"file_path"`
		WebPath  string `yaml:"web_path"`
	} `yaml:"static"`
}

type SMTP struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Unmarshal 解析 YAML 数据
func Unmarshal(filePath string) (*Config, error) {
	var cfg Config

	// 打开配置文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取文件内容
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
