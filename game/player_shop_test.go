package game

import (
	"math"
	"testing"

	"gucooing/lolo/protocol/excel"
)

func TestShopPurchaseCostsValidateQuantityAndCurrency(t *testing.T) {
	conf := &excel.ShopPoolConfigureItem{
		ShopCurrencyItem: []*excel.ShopCurrencyItems{
			{CurrencyID: 1, Price: 7},
			{CurrencyID: 2, Price: 0},
		},
	}
	costs, ok := shopPurchaseCosts(conf, 3)
	if !ok || len(costs) != 1 || costs[0].itemID != 1 || costs[0].amount != 21 {
		t.Fatalf("unexpected costs: %+v, valid=%v", costs, ok)
	}
	for _, times := range []uint32{0, maxShopBuyTimes + 1, math.MaxUint32} {
		if _, ok := shopPurchaseCosts(conf, times); ok {
			t.Fatalf("buy times %d must be rejected", times)
		}
	}
	conf.ShopCurrencyItem[0].Price = -1
	if _, ok := shopPurchaseCosts(conf, 1); ok {
		t.Fatal("negative price must be rejected")
	}
}

func TestShopPurchaseCostsDoNotOverflowInt32Multiplication(t *testing.T) {
	conf := &excel.ShopPoolConfigureItem{
		ShopCurrencyItem: []*excel.ShopCurrencyItems{
			{CurrencyID: 1, Price: math.MaxInt32},
		},
	}
	costs, ok := shopPurchaseCosts(conf, maxShopBuyTimes)
	if !ok || len(costs) != 1 {
		t.Fatalf("valid large purchase failed: %+v, valid=%v", costs, ok)
	}
	want := int64(math.MaxInt32) * int64(maxShopBuyTimes)
	if costs[0].amount != want {
		t.Fatalf("cost = %d, want %d", costs[0].amount, want)
	}
}
