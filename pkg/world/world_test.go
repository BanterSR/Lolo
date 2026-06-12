package world

import (
	"testing"
	"time"
)

func TestCalcWeather(t *testing.T) {
	var (
		seed int64 = time.Now().Unix()
		day  int64 = 0
	)
	for i := int64(0); i < 24; i++ {
		weather := CalcWeather(seed, day, i)
		t.Logf("weather:%s", weather.String())
	}
}
