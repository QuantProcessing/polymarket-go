package arb

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// OrderbookRecorder writes orderbook snapshots to CSV files for backtesting/review.
// Each market gets its own CSV file in the configured output directory.
type OrderbookRecorder struct {
	dir    string
	logger *zap.SugaredLogger

	mu     sync.Mutex
	file   *os.File
	writer *csv.Writer
	market string
	count  int64
}

// NewOrderbookRecorder creates a recorder that writes to the given directory.
func NewOrderbookRecorder(dir string, logger *zap.SugaredLogger) *OrderbookRecorder {
	return &OrderbookRecorder{
		dir:    dir,
		logger: logger,
	}
}

// StartMarket begins recording for a new market, creating a new CSV file.
func (r *OrderbookRecorder) StartMarket(market string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Close previous file if any
	r.closeInternal()

	// Ensure directory exists
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return fmt.Errorf("failed to create recorder dir: %w", err)
	}

	// Create CSV file: <dir>/<market>_<date>.csv
	filename := fmt.Sprintf("%s_%s.csv", market, time.Now().Format("20060102"))
	path := filepath.Join(r.dir, filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}

	w := csv.NewWriter(f)

	// Write header if file is new (size == 0)
	info, _ := f.Stat()
	if info.Size() == 0 {
		if err := w.Write([]string{
			"timestamp", "market",
			"yes_bid", "yes_bid_size", "yes_ask", "yes_ask_size",
			"no_bid", "no_bid_size", "no_ask", "no_ask_size",
		}); err != nil {
			f.Close()
			return fmt.Errorf("failed to write CSV header: %w", err)
		}
		w.Flush()
	}

	r.file = f
	r.writer = w
	r.market = market
	r.count = 0

	r.logger.Infow("Orderbook recorder started", "file", path)
	return nil
}

// Record writes a single orderbook snapshot to CSV.
func (r *OrderbookRecorder) Record(ob *OrderbookState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.writer == nil {
		return
	}

	var yesBid, yesBidSize, yesAsk, yesAskSize float64
	var noBid, noBidSize, noAsk, noAskSize float64

	if len(ob.YESBids) > 0 {
		yesBid = ob.YESBids[0].Price
		yesBidSize = ob.YESBids[0].Size
	}
	if len(ob.YESAsks) > 0 {
		yesAsk = ob.YESAsks[0].Price
		yesAskSize = ob.YESAsks[0].Size
	}
	if len(ob.NOBids) > 0 {
		noBid = ob.NOBids[0].Price
		noBidSize = ob.NOBids[0].Size
	}
	if len(ob.NOAsks) > 0 {
		noAsk = ob.NOAsks[0].Price
		noAskSize = ob.NOAsks[0].Size
	}

	r.writer.Write([]string{
		time.Now().Format(time.RFC3339Nano),
		r.market,
		fmt.Sprintf("%.4f", yesBid), fmt.Sprintf("%.2f", yesBidSize),
		fmt.Sprintf("%.4f", yesAsk), fmt.Sprintf("%.2f", yesAskSize),
		fmt.Sprintf("%.4f", noBid), fmt.Sprintf("%.2f", noBidSize),
		fmt.Sprintf("%.4f", noAsk), fmt.Sprintf("%.2f", noAskSize),
	})

	r.count++
	// Flush every 50 records for durability
	if r.count%50 == 0 {
		r.writer.Flush()
	}
}

// StopMarket flushes and closes the current CSV file.
func (r *OrderbookRecorder) StopMarket() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeInternal()
}

func (r *OrderbookRecorder) closeInternal() {
	if r.writer != nil {
		r.writer.Flush()
		r.writer = nil
	}
	if r.file != nil {
		r.file.Close()
		r.file = nil
		r.logger.Infow("Orderbook recorder stopped", "records", r.count)
	}
}
