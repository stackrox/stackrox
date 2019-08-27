package dump

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	cpuProfileDuration = 30 * time.Second

	prometheus = "http://localhost:9090/metrics" // This should be inferred after another PR goes in
)

var (
	log = logging.LoggerForModule()

	client = &http.Client{
		Timeout: time.Second * 5,
	}
)

func getPrometheusMetrics(zipWriter *zip.Writer, name string) error {
	resp, err := client.Get(prometheus)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	w, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(bytes)
	return err
}

func getMemory(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("heap.tar.gz")
	if err != nil {
		return err
	}
	return pprof.WriteHeapProfile(w)
}

func getCPU(zipWriter *zip.Writer, duration time.Duration) error {
	w, err := zipWriter.Create("cpu.tar.gz")
	if err != nil {
		return err
	}
	if err := pprof.StartCPUProfile(w); err != nil {
		return err
	}
	time.Sleep(duration)
	pprof.StopCPUProfile()
	return nil
}

func getGoroutines(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("goroutine.txt")
	if err != nil {
		return err
	}
	p := pprof.Lookup("goroutine")
	return p.WriteTo(w, 2)
}

func getLogs(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("central.log")
	if err != nil {
		return err
	}

	logFile, err := os.Open(logging.LoggingPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, logFile)
	return err
}

// DebugHandler is an HTTP handler that outputs debugging information
func DebugHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		withLogs := true
		for _, p := range r.URL.Query()["logs"] {
			v, err := strconv.ParseBool(p)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "invalid log value: %q", p)
				return
			}
			withLogs = v
		}

		filename := time.Now().Format("stackrox_debug_2006_01_02_15_04_05.zip")

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

		zipWriter := zip.NewWriter(w)
		defer utils.IgnoreError(zipWriter.Close)

		if err := getPrometheusMetrics(zipWriter, "metrics-1"); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := getMemory(zipWriter); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := getGoroutines(zipWriter); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := getCPU(zipWriter, cpuProfileDuration); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := getPrometheusMetrics(zipWriter, "metrics-2"); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if withLogs {
			if err := getLogs(zipWriter); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

	}
}
