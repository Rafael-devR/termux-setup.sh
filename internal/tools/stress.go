package tools

import (
	"context"
	"math/rand/v2"
	"runtime"
	"sync"
	"time"
)

// StressSample é uma leitura durante o teste de estresse.
type StressSample struct {
	Second  int
	TempC   float64 // 0 = sem leitura
	Battery int
}

// Stress carrega todos os núcleos por d e lê a temperatura a cada 2 s.
// No modo teste não gasta CPU: só simula a curva.
func (t *Toolbox) Stress(ctx context.Context, d time.Duration, emit func(StressSample)) {
	cctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	if !t.Fake {
		var wg sync.WaitGroup
		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				x := 1.0
				for cctx.Err() == nil {
					for j := 0; j < 200000; j++ {
						x = x*1.0000001 + 0.5
					}
				}
				_ = x
			}()
		}
		defer wg.Wait()
	}
	start := time.Now()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		s := StressSample{Second: int(time.Since(start).Seconds())}
		if t.Fake {
			s.TempC = 33 + float64(s.Second)*0.25 + rand.Float64()
			s.Battery = 64
		} else if b, err := t.ReadBattery(cctx); err == nil {
			s.TempC, s.Battery = b.Temp, b.Percent
		} else if th, ok := t.ReadThermal(); ok {
			s.TempC = th
		}
		emit(s)
		select {
		case <-cctx.Done():
			return
		case <-tick.C:
		}
	}
}
