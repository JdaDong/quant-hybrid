package main

import "testing"

func testControlPlane() *ControlPlane {
	return &ControlPlane{
		positions:   map[string]int64{"DEMO": 10},
		maxQty:      100,
		maxPos:      1_000,
		maxNotional: 100_000,
	}
}

func TestValidateAcceptsOrder(t *testing.T) {
	cp := testControlPlane()
	order := Order{Symbol: "DEMO", Side: "BUY", Quantity: 20, LimitPrice: 100}
	if err := cp.validate(order); err != nil {
		t.Fatalf("valid order rejected: %v", err)
	}
}

func TestValidateRejectsQuantityLimit(t *testing.T) {
	cp := testControlPlane()
	order := Order{Symbol: "DEMO", Side: "BUY", Quantity: 101, LimitPrice: 100}
	if err := cp.validate(order); err == nil {
		t.Fatal("quantity limit was not enforced")
	}
}

func TestValidateRejectsPositionLimit(t *testing.T) {
	cp := testControlPlane()
	cp.positions["DEMO"] = 995
	order := Order{Symbol: "DEMO", Side: "BUY", Quantity: 10, LimitPrice: 100}
	if err := cp.validate(order); err == nil {
		t.Fatal("position limit was not enforced")
	}
}
