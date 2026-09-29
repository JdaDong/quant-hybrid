package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Order struct {
	OrderID    string  `json:"order_id"`
	Symbol     string  `json:"symbol"`
	Side       string  `json:"side"`
	Quantity   int64   `json:"quantity"`
	LimitPrice float64 `json:"limit_price"`
}

type ControlPlane struct {
	engineIn    *bufio.Writer
	engineMu    sync.Mutex
	stateMu     sync.RWMutex
	positions   map[string]int64
	cash        float64
	events      []map[string]any
	sequence    uint64
	maxQty      int64
	maxPos      int64
	maxNotional float64
}

func newControlPlane(enginePath string) (*ControlPlane, error) {
	cmd := exec.Command(enginePath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start execution core: %w", err)
	}

	cp := &ControlPlane{
		engineIn:    bufio.NewWriter(stdin),
		positions:   make(map[string]int64),
		cash:        1_000_000,
		maxQty:      100,
		maxPos:      1_000,
		maxNotional: 100_000,
	}
	go cp.consumeEvents(stdout)
	return cp, nil
}

func (cp *ControlPlane) consumeEvents(stdout interface{ Read([]byte) (int, error) }) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var event map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			log.Printf("invalid execution event: %v", err)
			continue
		}
		cp.stateMu.Lock()
		cp.events = append(cp.events, event)
		if len(cp.events) > 100 {
			cp.events = cp.events[len(cp.events)-100:]
		}
		if typ, _ := event["type"].(string); typ == "fill" {
			symbol, _ := event["symbol"].(string)
			position, _ := event["position"].(float64)
			cash, _ := event["cash"].(float64)
			cp.positions[symbol] = int64(position)
			cp.cash = cash
		}
		cp.stateMu.Unlock()
	}
	if err := scanner.Err(); err != nil {
		log.Printf("execution core disconnected: %v", err)
	}
}

func (cp *ControlPlane) validate(order Order) error {
	if order.Symbol == "" || len(order.Symbol) > 32 {
		return errors.New("symbol is required and must be <= 32 chars")
	}
	order.Side = strings.ToUpper(order.Side)
	if order.Side != "BUY" && order.Side != "SELL" {
		return errors.New("side must be BUY or SELL")
	}
	if order.Quantity <= 0 || order.Quantity > cp.maxQty {
		return fmt.Errorf("quantity must be in [1,%d]", cp.maxQty)
	}
	if order.LimitPrice <= 0 || float64(order.Quantity)*order.LimitPrice > cp.maxNotional {
		return fmt.Errorf("notional must be <= %.2f", cp.maxNotional)
	}
	cp.stateMu.RLock()
	current := cp.positions[order.Symbol]
	cp.stateMu.RUnlock()
	delta := order.Quantity
	if order.Side == "SELL" {
		delta = -delta
	}
	if abs(current+delta) > cp.maxPos {
		return fmt.Errorf("absolute position must be <= %d", cp.maxPos)
	}
	return nil
}

func abs(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func (cp *ControlPlane) submit(order Order) error {
	cp.engineMu.Lock()
	defer cp.engineMu.Unlock()
	payload := struct {
		Type string `json:"type"`
		Order
	}{Type: "order", Order: order}
	if err := json.NewEncoder(cp.engineIn).Encode(payload); err != nil {
		return err
	}
	return cp.engineIn.Flush()
}

func (cp *ControlPlane) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "control_plane": "go", "execution_core": "cpp"})
}

func (cp *ControlPlane) createOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return
	}
	order.Side = strings.ToUpper(order.Side)
	if order.OrderID == "" {
		seq := atomic.AddUint64(&cp.sequence, 1)
		order.OrderID = "go-" + strconv.FormatUint(seq, 10)
	}
	if err := cp.validate(order); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := cp.submit(order); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "execution core unavailable"})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"order_id": order.OrderID, "status": "accepted_by_control_plane"})
}

func (cp *ControlPlane) snapshot(w http.ResponseWriter, _ *http.Request) {
	cp.stateMu.RLock()
	defer cp.stateMu.RUnlock()
	positions := make(map[string]int64, len(cp.positions))
	for symbol, quantity := range cp.positions {
		positions[symbol] = quantity
	}
	writeJSON(w, http.StatusOK, map[string]any{"cash": cp.cash, "positions": positions, "events": cp.events})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func main() {
	enginePath := flag.String("engine", "./bin/execution_core", "path to C++ execution core")
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()

	cp, err := newControlPlane(*enginePath)
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", cp.health)
	mux.HandleFunc("/orders", cp.createOrder)
	mux.HandleFunc("/snapshot", cp.snapshot)
	server := &http.Server{Addr: *addr, Handler: logging(mux)}
	log.Printf("Go control plane listening on http://%s", *addr)
	log.Printf("risk limits: max_qty=%d max_abs_position=%d max_notional=%.2f", cp.maxQty, cp.maxPos, cp.maxNotional)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
