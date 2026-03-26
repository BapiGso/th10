package game

// Config 全局游戏配置（由 Option 场景修改，启动时加载/退出时保存）
type Config struct {
	BGMVolume float64 // 0.0 ~ 1.0
	SEVolume  float64 // 0.0 ~ 1.0
	Hint      bool    // 开启 hint 提示
}

func DefaultConfig() *Config {
	return &Config{
		BGMVolume: 0.8,
		SEVolume:  0.8,
		Hint:      true,
	}
}
