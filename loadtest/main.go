// loadtest is a concurrent load-testing tool for the wallet HTTP API.
//
// Usage:
//
//	go run ./loadtest [flags]
//
// Flags:
//
//	-addr      Base URL of the wallet service (default: http://localhost:8080)
//	-wallets   Number of wallets to pre-create (default: 20)
//	-workers   Number of concurrent goroutines (default: 50)
//	-duration  Test duration, e.g. 30s, 1m (default: 30s)
//	-deposit   Initial deposit per wallet in cents (default: 100000)
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// ── CLI flags ────────────────────────────────────────────────────────────────

var (
	addr        = flag.String("addr", "http://localhost:8080", "wallet service base URL")
	numWallets  = flag.Int("wallets", 20, "number of wallets to pre-create")
	numWorkers  = flag.Int("workers", 50, "concurrent workers")
	duration    = flag.Duration("duration", 30*time.Second, "test duration")
	initDeposit = flag.Int64("deposit", 100_000, "initial deposit per wallet (cents)")
)

// ── counters ─────────────────────────────────────────────────────────────────

type counters struct {
	total   atomic.Int64
	success atomic.Int64
	failed  atomic.Int64
	latSum  atomic.Int64 // nanoseconds
}

// ── HTTP helpers ─────────────────────────────────────────────────────────────

var client = &http.Client{Timeout: 10 * time.Second}

func post(url string, body any) (int, []byte, error) {
	b, _ := json.Marshal(body)
	resp, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data, nil
}

func get(url string) (int, []byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data, nil
}

// ── wallet API ────────────────────────────────────────────────────────────────

func createWallet() (string, error) {
	code, data, err := post(*addr+"/wallets", nil)
	if err != nil {
		return "", err
	}
	if code != 201 {
		return "", fmt.Errorf("create wallet: status %d", code)
	}
	var w struct {
		ID string `json:"id"`
	}
	json.Unmarshal(data, &w)
	return w.ID, nil
}

func deposit(id string, amount int64) error {
	code, _, err := post(fmt.Sprintf("%s/wallets/%s/deposit", *addr, id), map[string]int64{"amount": amount})
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("deposit: status %d", code)
	}
	return nil
}

func transfer(from, to string, amount int64) error {
	code, _, err := post(*addr+"/wallets/transfer", map[string]any{
		"from_id": from,
		"to_id":   to,
		"amount":  amount,
	})
	if err != nil {
		return err
	}
	if code != 204 && code != 422 { // 422 = insufficient funds, expected under load
		return fmt.Errorf("transfer: status %d", code)
	}
	return nil
}

func getWallet(id string) error {
	code, _, err := get(fmt.Sprintf("%s/wallets/%s", *addr, id))
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("get wallet: status %d", code)
	}
	return nil
}

// ── worker ────────────────────────────────────────────────────────────────────

func worker(wallets []string, c *counters, stop <-chan struct{}, rng *rand.Rand) {
	n := len(wallets)
	for {
		select {
		case <-stop:
			return
		default:
		}

		start := time.Now()
		var err error

		// Randomly pick an operation: 20% get, 30% deposit, 50% transfer
		op := rng.Intn(10)
		switch {
		case op < 2: // get
			err = getWallet(wallets[rng.Intn(n)])
		case op < 5: // deposit
			err = deposit(wallets[rng.Intn(n)], int64(rng.Intn(1000)+1))
		default: // transfer
			from := rng.Intn(n)
			to := (from + 1 + rng.Intn(n-1)) % n
			err = transfer(wallets[from], wallets[to], int64(rng.Intn(500)+1))
		}

		lat := time.Since(start).Nanoseconds()
		c.total.Add(1)
		c.latSum.Add(lat)
		if err != nil {
			c.failed.Add(1)
		} else {
			c.success.Add(1)
		}
	}
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	flag.Parse()

	fmt.Printf("Wallet Load Test\n")
	fmt.Printf("  addr=%s  wallets=%d  workers=%d  duration=%s\n\n",
		*addr, *numWallets, *numWorkers, *duration)

	// ── setup: create wallets and fund them ──────────────────────────────────
	fmt.Printf("Creating %d wallets...\n", *numWallets)
	wallets := make([]string, 0, *numWallets)
	for i := 0; i < *numWallets; i++ {
		id, err := createWallet()
		if err != nil {
			fmt.Fprintf(os.Stderr, "setup error: %v\n", err)
			os.Exit(1)
		}
		if err := deposit(id, *initDeposit); err != nil {
			fmt.Fprintf(os.Stderr, "setup deposit error: %v\n", err)
			os.Exit(1)
		}
		wallets = append(wallets, id)
	}
	fmt.Printf("Setup done. Starting %d workers for %s...\n\n", *numWorkers, *duration)

	// ── run ──────────────────────────────────────────────────────────────────
	var c counters
	stop := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i)))
		go func() {
			defer wg.Done()
			worker(wallets, &c, stop, rng)
		}()
	}

	// progress ticker
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			t := c.total.Load()
			if t == 0 {
				continue
			}
			avgMs := float64(c.latSum.Load()) / float64(t) / 1e6
			fmt.Printf("  progress: total=%-8d ok=%-8d fail=%-6d avg_lat=%.2fms\n",
				t, c.success.Load(), c.failed.Load(), avgMs)
		}
	}()

	time.Sleep(*duration)
	close(stop)
	ticker.Stop()
	wg.Wait()

	// ── report ───────────────────────────────────────────────────────────────
	total := c.total.Load()
	success := c.success.Load()
	failed := c.failed.Load()
	elapsed := duration.Seconds()

	var avgMs float64
	if total > 0 {
		avgMs = float64(c.latSum.Load()) / float64(total) / 1e6
	}

	fmt.Printf("\n── Results ──────────────────────────────────────\n")
	fmt.Printf("  Duration   : %s\n", *duration)
	fmt.Printf("  Workers    : %d\n", *numWorkers)
	fmt.Printf("  Total reqs : %d\n", total)
	fmt.Printf("  Success    : %d (%.1f%%)\n", success, 100*float64(success)/float64(total))
	fmt.Printf("  Failed     : %d\n", failed)
	fmt.Printf("  Throughput : %.1f req/s\n", float64(total)/elapsed)
	fmt.Printf("  Avg latency: %.2f ms\n", avgMs)
}
