package world

import (
	"gucooing/lolo/protocol/proto"
	"time"
)

func CalcWeather(seed int64, day, hour int64) proto.WeatherType {
	slot := day*24 + hour

	x := uint64(seed) + uint64(slot)*0x9e3779b97f4a7c15
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31

	if x%100 < getWeatherBase() {
		return proto.WeatherType_WeatherType_Rainy
	}
	return proto.WeatherType_WeatherType_Sunny
}

// 下雨的概率
func getWeatherBase() uint64 {
	switch time.Now().Hour() {
	case 3, 4, 5: // 春
		return 35
	case 6, 7, 8: // 夏，雨季
		return 55
	case 9, 10, 11: // 秋
		return 20
	default: // 冬
		return 15
	}
}
