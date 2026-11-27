package model

import (
	"testing"

	"gucooing/lolo/protocol/excel"
)

func TestFullPickAvailabilityRequiresProgressAndUnclaimedState(t *testing.T) {
	conf := &excel.GachaInfoConfigure{OptionalNum: 10}
	info := &GachaInfo{GachaTimes: 9}
	if fullPickAvailable(conf, info) {
		t.Fatal("full pick must not be available before the configured pull count")
	}
	info.GachaTimes = 10
	if !fullPickAvailable(conf, info) {
		t.Fatal("full pick must be available at the configured pull count")
	}
	info.FullPickRewarded = true
	if fullPickAvailable(conf, info) {
		t.Fatal("claimed full pick must not be available again")
	}
}

func TestFullPickRejectsMissingConfiguration(t *testing.T) {
	if fullPickAvailable(nil, &GachaInfo{GachaTimes: 100}) {
		t.Fatal("missing configuration must not enable full pick")
	}
	if fullPickAvailable(&excel.GachaInfoConfigure{}, &GachaInfo{GachaTimes: 100}) {
		t.Fatal("zero optional count must not enable full pick")
	}
}
