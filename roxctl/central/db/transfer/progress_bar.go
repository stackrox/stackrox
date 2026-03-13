package transfer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"golang.org/x/term"
)

const (
	progressBarWidth       = 120
	nonTerminalRefreshRate = 1 * time.Minute
)

type singleCounterDecorator struct {
	decor.WC
	fmt string
}

func newSingleCounterDecorator(fmt string) decor.Decorator {
	wc := decor.WC{}
	wc.Init()
	return &singleCounterDecorator{
		WC:  wc,
		fmt: fmt,
	}
}

func (d *singleCounterDecorator) Decor(st decor.Statistics) (string, int) {
	str := fmt.Sprintf(d.fmt, decor.SizeB1024(st.Current))
	return d.WC.Format(str)
}

type unknownTotalSizeFiller struct {
	tick int
}

func (f *unknownTotalSizeFiller) Fill(w io.Writer, st decor.Statistics) error {
	f.tick++

	effectiveWidth := st.AvailableWidth - 2

	arrowWidth := 5
	arrowSpace := 10
	total := arrowWidth + arrowSpace

	bar := strings.Builder{}
	_, _ = bar.WriteRune('[')
	for i := 0; i < effectiveWidth; i++ {
		if i > f.tick || (f.tick-i)%total >= arrowWidth {
			_, _ = bar.WriteRune('-')
		} else {
			_, _ = bar.WriteRune('>')
		}
	}
	_, _ = bar.WriteRune(']')
	_, err := w.Write([]byte(bar.String()))
	return err
}

func createProgressBars(_ context.Context, name string, totalSize int64) (*mpb.Bar, func()) {
	outFile := os.Stderr //nolint:forbidigo // TODO(ROX-13473)

	opts := []mpb.ContainerOption{
		mpb.WithWidth(progressBarWidth),
		mpb.WithOutput(outFile),
	}

	shutdownSig := concurrency.NewSignal()

	if !term.IsTerminal(int(outFile.Fd())) {
		refreshC := make(chan interface{}, 1)
		refreshC <- time.Now() // first tick right away
		shutdownNotifyC := make(chan interface{})
		shutdownC := shutdownSig.Done()

		go func() {
			t := time.NewTicker(nonTerminalRefreshRate)
			defer t.Stop()

			for {
				select {
				case tick := <-t.C:
					refreshC <- tick
				case <-shutdownC:
					shutdownC = nil
					refreshC <- time.Now()
					t = time.NewTicker(100 * time.Millisecond)
					defer t.Stop()
				case <-shutdownNotifyC:
					return
				}
			}
		}()

		opts = append(opts, mpb.WithManualRefresh(refreshC), mpb.WithShutdownNotifier(shutdownNotifyC))
	}

	progressBars := mpb.New(opts...)

	var progressBar *mpb.Bar
	var counterDecorator decor.Decorator

	appendedDecorators := []decor.Decorator{
		decor.AverageSpeed(decor.SizeB1024(0), "% 11.1f"),
		decor.Name(fmt.Sprintf(" %s ", name)),
	}

	if totalSize == 0 {
		filler := &unknownTotalSizeFiller{}
		counterDecorator = newSingleCounterDecorator("% 10.1f")
		var err error
		progressBar, err = progressBars.Add(totalSize, filler,
			mpb.PrependDecorators(counterDecorator),
			mpb.AppendDecorators(appendedDecorators...),
		)
		if err != nil {
			// Should not happen unless progressBars.Wait() was already called
			panic(err)
		}
	} else {
		counterDecorator = decor.CountersKibiByte("% 10.1f / % 10.1f")
		appendedDecorators = append(appendedDecorators, decor.AverageETA(decor.ET_STYLE_MMSS))
		progressBar = progressBars.AddBar(totalSize,
			mpb.PrependDecorators(counterDecorator),
			mpb.AppendDecorators(appendedDecorators...),
		)
	}

	shutdownFunc := func() {
		if totalSize == 0 && !progressBar.Completed() {
			progressBar.Abort(false)
		}
		shutdownSig.Signal()
		progressBars.Wait()
	}

	return progressBar, shutdownFunc
}
