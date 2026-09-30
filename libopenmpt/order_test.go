package libopenmpt_test

import (
	"testing"

	"github.com/Lakelimbo/nezumi/internal/utils"
)

func TestOrderPatternPlaybackPosition(t *testing.T) {
	mod := openDemo(t, utils.DemoRealization)

	if got := mod.OrderPattern(0); got != mod.CurrentPattern() {
		t.Errorf(
			"OrderPattern(0) is %d, want %d: the module opens on order 0",
			got,
			mod.CurrentPattern(),
		)
	}

	const (
		order   = 14
		pattern = 12
	)

	if got := mod.OrderPattern(order); got != pattern {
		t.Errorf("OrderPattern(%d) is %d, want %d", order, got, pattern)
	}

	if err := mod.SeekOrderRow(order, 0); err != nil {
		t.Fatalf("SeekOrderRow(%d, 0): %v", order, err)
	}

	if got, want := mod.OrderPattern(order), mod.CurrentPattern(); got != want {
		t.Errorf("OrderPattern(%d) is %d after seeking, want %d", order, got, want)
	}
}

func TestOrderPatternAnswersForPlayableOrders(t *testing.T) {
	mod := openDemo(t, utils.DemoRealization)

	total := mod.NumOrders()
	if total <= 0 {
		t.Fatalf("NumOrders is %d, want a positive amount", total)
	}

	playable := 0

	for order := range total {
		pattern := mod.OrderPattern(order)

		if mod.IsOrderPlayable(order) != (pattern >= 0) {
			t.Errorf("order %d: IsOrderPlayable is %t but OrderPattern is %d",
				order, mod.IsOrderPlayable(order), pattern)
		}

		if mod.IsOrderPlayable(order) {
			playable++

			if pattern >= mod.NumPatterns() {
				t.Errorf("order %d holds pattern %d, but module has %d patterns",
					order, pattern, mod.NumPatterns())
			}
		}
	}

	if playable == 0 {
		t.Errorf("none of the %d orders in the module is playable", total)
	}
}

func TestOrderQueriesRejectPositionOutsideSequence(t *testing.T) {
	mod := openDemo(t, utils.DemoRealization)

	for _, order := range []int{
		-1,
		mod.NumOrders(),
		mod.NumOrders() + 100,
	} {
		if got := mod.OrderPattern(order); got != -1 {
			t.Errorf("OrderPattern(%d) is %d, want -1", order, got)
		}

		if mod.IsOrderPlayable(order) {
			t.Errorf("IsOrderPlayable(%d) is true, want false", order)
		}
	}
}
