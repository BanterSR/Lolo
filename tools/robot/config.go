package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Duration 是可从 JSON 字符串("30s")或纳秒数字解析的时长类型。
type Duration time.Duration

func (d Duration) D() time.Duration { return time.Duration(d) }

func (d Duration) String() string { return time.Duration(d).String() }

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		if s == "" {
			*d = 0
			return nil
		}
		v, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		*d = Duration(v)
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*d = Duration(n)
	return nil
}

// Stage 是 profile 模式的一个负载阶段。
type Stage struct {
	CCU  int      `json:"ccu"`
	Hold Duration `json:"hold"`
}

// Config 是压测工具的完整配置，可由 -config JSON 文件与命令行 flag 共同决定。
type Config struct {
	Gate string `json:"gate"` // 网关 TCP 地址
	Sdk  string `json:"sdk"`  // SDK HTTP 基地址

	Mode     string `json:"mode"`     // routine | endurance | profile
	Scenario string `json:"scenario"` // login | steady | scene

	CCU      int      `json:"ccu"`      // 目标并发 bot 数
	Ramp     Duration `json:"ramp"`     // 爬坡时长
	Duration Duration `json:"duration"` // 保持时长

	// profile
	Pattern  string   `json:"pattern"`  // step | wave | spike（无 stages 时生效）
	Peak     int      `json:"peak"`     // 曲线峰值并发
	Base     int      `json:"base"`     // 曲线基线并发
	Period   Duration `json:"period"`   // wave 周期
	Steps    int      `json:"steps"`    // step 阶梯数
	StepHold Duration `json:"stepHold"` // step/spike 每段时长
	Spike    Duration `json:"spike"`    // spike 峰值持续时长
	Loop     bool     `json:"loop"`     // 是否循环
	Stages   []Stage  `json:"stages"`   // 显式阶段列表

	// 账号
	Prefix   string `json:"prefix"`   // 用户名前缀，账号 = prefix+id
	Password string `json:"password"` // 账号密码

	// 场景调优
	Ping         Duration `json:"ping"`         // ping 间隔
	Action       Duration `json:"action"`       // 场景动作间隔
	Report       Duration `json:"report"`       // 上报间隔
	RetryBackoff Duration `json:"retryBackoff"` // 登录失败后重试前的退避时长

	// 观测
	Web   string `json:"web"`   // web 仪表盘监听地址，空=关闭，如 :8090
	Pprof string `json:"pprof"` // 服务端 pprof 基地址，空=沿用 -sdk

	source string // 实际配置来源(内部使用,不序列化)
}

// configCandidates 返回配置文件候选路径:显式 -config 只用其本身,
// 否则依次尝试工作目录与程序所在目录下的 config.json。
func configCandidates(configPath string, explicit bool) []string {
	if explicit {
		return []string{configPath}
	}
	paths := []string{configPath}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "config.json"))
	}
	return paths
}

func defaultConfig() *Config {
	return &Config{
		Gate:     "127.0.0.1:11000",
		Sdk:      "http://127.0.0.1:8080",
		Mode:     "routine",
		Scenario: "steady",
		CCU:      100,
		Ramp:     Duration(10 * time.Second),
		Duration: Duration(60 * time.Second),
		Base:     0,
		Steps:    5,
		Prefix:   "robot_",
		Password: "robot123456",
		Ping:         Duration(15 * time.Second),
		Action:       Duration(2 * time.Second),
		Report:       Duration(1 * time.Second),
		RetryBackoff: Duration(2 * time.Second),
	}
}

// LoadConfig 解析 flag、可选叠加 JSON 配置文件，最后用显式 flag 覆盖。
func LoadConfig() (*Config, error) {
	cfg := defaultConfig()

	var configPath string
	gate := flag.String("gate", cfg.Gate, "网关 TCP 地址")
	sdk := flag.String("sdk", cfg.Sdk, "SDK HTTP 基地址")
	mode := flag.String("mode", cfg.Mode, "模式: routine|endurance|profile")
	scenario := flag.String("scenario", cfg.Scenario, "场景: login|steady|scene")
	ccu := flag.Int("ccu", cfg.CCU, "目标并发 bot 数")
	ramp := flag.String("ramp", "", "爬坡时长, 如 30s")
	duration := flag.String("duration", "", "保持时长, 如 5m")
	pattern := flag.String("pattern", cfg.Pattern, "profile 内置曲线: step|wave|spike")
	peak := flag.Int("peak", cfg.Peak, "曲线峰值并发")
	base := flag.Int("base", cfg.Base, "曲线基线并发")
	period := flag.String("period", "", "wave 周期, 如 2m")
	steps := flag.Int("steps", cfg.Steps, "step 阶梯数")
	stepHold := flag.String("step-hold", "", "step/spike 每段时长")
	spike := flag.String("spike", "", "spike 峰值持续时长")
	loop := flag.Bool("loop", cfg.Loop, "profile 是否循环")
	prefix := flag.String("prefix", cfg.Prefix, "账号用户名前缀")
	password := flag.String("password", cfg.Password, "账号密码")
	ping := flag.String("ping", "", "ping 间隔, 如 15s")
	action := flag.String("action", "", "场景动作间隔, 如 2s")
	report := flag.String("report", "", "上报间隔, 如 1s")
	retryBackoff := flag.String("retry-backoff", "", "登录失败后重试前的退避, 如 2s")
	web := flag.String("web", cfg.Web, "web 仪表盘地址, 如 :8090, 空则关闭")
	pprofBase := flag.String("pprof", cfg.Pprof, "服务端 pprof 基地址, 空则沿用 -sdk")
	flag.StringVar(&configPath, "config", "config.json", "JSON 配置文件路径(默认读取工作目录/程序目录下的 config.json)")
	flag.Parse()

	set := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

	// 定位配置文件:未显式 -config 时,依次在工作目录与程序所在目录找 config.json
	loadPath := ""
	for _, p := range configCandidates(configPath, set["config"]) {
		if _, err := os.Stat(p); err == nil {
			loadPath = p
			break
		}
	}
	switch {
	case loadPath != "":
		data, err := os.ReadFile(loadPath)
		if err != nil {
			return nil, fmt.Errorf("读取配置文件 %s: %w", loadPath, err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件 %s: %w", loadPath, err)
		}
		if abs, err := filepath.Abs(loadPath); err == nil {
			loadPath = abs
		}
		cfg.source = loadPath
	case set["config"]:
		return nil, fmt.Errorf("指定的配置文件不存在: %s", configPath)
	default:
		cfg.source = "内置默认值(未找到 config.json)"
	}

	if set["gate"] {
		cfg.Gate = *gate
	}
	if set["sdk"] {
		cfg.Sdk = *sdk
	}
	if set["mode"] {
		cfg.Mode = *mode
	}
	if set["scenario"] {
		cfg.Scenario = *scenario
	}
	if set["ccu"] {
		cfg.CCU = *ccu
	}
	if set["pattern"] {
		cfg.Pattern = *pattern
	}
	if set["peak"] {
		cfg.Peak = *peak
	}
	if set["base"] {
		cfg.Base = *base
	}
	if set["steps"] {
		cfg.Steps = *steps
	}
	if set["loop"] {
		cfg.Loop = *loop
	}
	if set["prefix"] {
		cfg.Prefix = *prefix
	}
	if set["password"] {
		cfg.Password = *password
	}
	if set["web"] {
		cfg.Web = *web
	}
	if set["pprof"] {
		cfg.Pprof = *pprofBase
	}

	durFlags := []struct {
		name string
		val  string
		dst  *Duration
	}{
		{"ramp", *ramp, &cfg.Ramp},
		{"duration", *duration, &cfg.Duration},
		{"period", *period, &cfg.Period},
		{"step-hold", *stepHold, &cfg.StepHold},
		{"spike", *spike, &cfg.Spike},
		{"ping", *ping, &cfg.Ping},
		{"action", *action, &cfg.Action},
		{"report", *report, &cfg.Report},
		{"retry-backoff", *retryBackoff, &cfg.RetryBackoff},
	}
	for _, f := range durFlags {
		if !set[f.name] {
			continue
		}
		v, err := time.ParseDuration(f.val)
		if err != nil {
			return nil, fmt.Errorf("-%s 时长解析失败: %w", f.name, err)
		}
		*f.dst = Duration(v)
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.Gate == "" || c.Sdk == "" {
		return fmt.Errorf("gate 与 sdk 地址不能为空")
	}
	switch c.Scenario {
	case "login", "steady", "scene":
	default:
		return fmt.Errorf("未知 scenario: %q（login|steady|scene）", c.Scenario)
	}
	switch c.Mode {
	case "routine", "endurance":
		if c.CCU <= 0 {
			return fmt.Errorf("%s 模式需要 -ccu > 0", c.Mode)
		}
	case "profile":
		if len(c.Stages) == 0 && c.Pattern == "" {
			return fmt.Errorf("profile 模式需要 -config 阶段列表或 -pattern step|wave|spike")
		}
	default:
		return fmt.Errorf("未知 mode: %q（routine|endurance|profile）", c.Mode)
	}
	return nil
}
