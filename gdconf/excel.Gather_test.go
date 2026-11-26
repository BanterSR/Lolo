package gdconf

import (
	"math"
	"testing"

	"gucooing/lolo/protocol/excel"
)

func TestGatherRuleLimitsMapsAllSixTypes(t *testing.T) {
	rule := &excel.GatherRuleConfigure{
		ID:                      1,
		WoodNum:                 11,
		FlowerNum:               12,
		StoneNum:                13,
		ButterflyNum:            14,
		WaterNum:                15,
		PickNum:                 16,
		LuckyFinishwoodNum:      21,
		LuckyFinishflowerNum:    22,
		LuckyFinishstoneNum:     23,
		LuckyFinishbutterflyNum: 24,
		LuckyFinishwaterNum:     25,
		LuckyFinishpickNum:      0,
	}
	for gatherType, want := range [][2]uint32{
		{11, 21}, {12, 22}, {13, 23}, {14, 24}, {15, 25}, {16, 0},
	} {
		normal, lucky, ok := GatherRuleLimits(rule, uint32(gatherType))
		if !ok || normal != want[0] || lucky != want[1] {
			t.Fatalf("type %d: got (%d,%d,%v), want (%d,%d,true)",
				gatherType, normal, lucky, ok, want[0], want[1])
		}
	}
}

func TestGatherRuleLimitsRejectsMissingNegativeAndOutOfRange(t *testing.T) {
	if _, _, ok := GatherRuleLimits(nil, 0); ok {
		t.Fatal("nil rule accepted")
	}
	for _, gatherType := range []uint32{GatherTypeCount, math.MaxUint32} {
		if _, _, ok := GatherRuleLimits(new(excel.GatherRuleConfigure), gatherType); ok {
			t.Fatalf("out-of-range type %d accepted", gatherType)
		}
	}
	for _, rule := range []*excel.GatherRuleConfigure{
		{WoodNum: -1},
		{LuckyFinishwoodNum: -1},
	} {
		if _, _, ok := GatherRuleLimits(rule, 0); ok {
			t.Fatalf("negative limit accepted: %+v", rule)
		}
	}
}
