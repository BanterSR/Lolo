package main

import (
	"fmt"
	"math"
	"time"
)

// LoadFunc 返回给定运行时刻应维持的目标并发数；done=true 表示测试应结束。
// 三种模式最终都归约为一条随时间变化的目标曲线，交给 Pool 收敛。
type LoadFunc func(elapsed time.Duration) (target int, done bool)

func buildLoad(cfg *Config) (LoadFunc, error) {
	switch cfg.Mode {
	case "routine":
		return rampHold(cfg.CCU, cfg.Ramp.D(), cfg.Duration.D()), nil
	case "endurance":
		d := cfg.Duration.D()
		if d <= 0 {
			d = 6 * time.Hour
		}
		return rampHold(cfg.CCU, cfg.Ramp.D(), d), nil
	case "profile":
		return buildProfile(cfg)
	default:
		return nil, fmt.Errorf("未知 mode: %q", cfg.Mode)
	}
}

// rampHold 在 ramp 内线性升到 ccu，随后保持 hold，之后结束。
func rampHold(ccu int, ramp, hold time.Duration) LoadFunc {
	return func(elapsed time.Duration) (int, bool) {
		if ramp > 0 && elapsed < ramp {
			return int(float64(ccu) * float64(elapsed) / float64(ramp)), false
		}
		if elapsed >= ramp+hold {
			return 0, true
		}
		return ccu, false
	}
}

// stageLoad 依次在各阶段维持对应并发，走完后结束或循环。
func stageLoad(stages []Stage, loop bool) LoadFunc {
	var total time.Duration
	for _, s := range stages {
		total += s.Hold.D()
	}
	return func(elapsed time.Duration) (int, bool) {
		if total <= 0 || len(stages) == 0 {
			return 0, true
		}
		t := elapsed
		if loop {
			t = time.Duration(int64(elapsed) % int64(total))
		} else if elapsed >= total {
			return 0, true
		}
		var acc time.Duration
		for _, s := range stages {
			if t < acc+s.Hold.D() {
				return s.CCU, false
			}
			acc += s.Hold.D()
		}
		return stages[len(stages)-1].CCU, false
	}
}

// waveLoad 在 base..peak 之间按余弦波连续变化，周期为 period。
func waveLoad(base, peak int, period, dur time.Duration, loop bool) LoadFunc {
	if period <= 0 {
		period = time.Minute
	}
	return func(elapsed time.Duration) (int, bool) {
		if !loop && dur > 0 && elapsed >= dur {
			return 0, true
		}
		v := (1 - math.Cos(2*math.Pi*float64(elapsed)/float64(period))) / 2 // 0..1
		return base + int(float64(peak-base)*v), false
	}
}

func buildProfile(cfg *Config) (LoadFunc, error) {
	if len(cfg.Stages) > 0 {
		return stageLoad(cfg.Stages, cfg.Loop), nil
	}
	peak := cfg.Peak
	if peak <= 0 {
		peak = cfg.CCU
	}
	switch cfg.Pattern {
	case "step":
		steps := cfg.Steps
		if steps < 1 {
			steps = 5
		}
		hold := cfg.StepHold.D()
		if hold <= 0 {
			hold = time.Minute
		}
		stages := make([]Stage, 0, steps)
		for k := 1; k <= steps; k++ {
			ccu := cfg.Base + (peak-cfg.Base)*k/steps
			stages = append(stages, Stage{CCU: ccu, Hold: Duration(hold)})
		}
		return stageLoad(stages, cfg.Loop), nil
	case "spike":
		warm := cfg.StepHold.D()
		if warm <= 0 {
			warm = 30 * time.Second
		}
		spike := cfg.Spike.D()
		if spike <= 0 {
			spike = 10 * time.Second
		}
		stages := []Stage{
			{CCU: cfg.Base, Hold: Duration(warm)},
			{CCU: peak, Hold: Duration(spike)},
			{CCU: cfg.Base, Hold: Duration(warm)},
		}
		return stageLoad(stages, cfg.Loop), nil
	case "wave":
		return waveLoad(cfg.Base, peak, cfg.Period.D(), cfg.Duration.D(), cfg.Loop), nil
	default:
		return nil, fmt.Errorf("未知 pattern: %q（step|wave|spike）", cfg.Pattern)
	}
}
