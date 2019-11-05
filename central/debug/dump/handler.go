package dump

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

const (
	cpuProfileDuration = 30 * time.Second

	prometheus = "http://127.0.0.1:9090/metrics" // This should be inferred after another PR goes in
)

var (
	log = logging.LoggerForModule()

	client = &http.Client{
		Timeout: time.Second * 5,
	}
)

func init() {
	runtime.SetBlockProfileRate(10)
	runtime.SetMutexProfileFraction(10)
}

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

func getBlock(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("block.tar.gz")
	if err != nil {
		return err
	}
	p := pprof.Lookup("block")
	return p.WriteTo(w, 0)
}

func getMutex(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("mutex.tar.gz")
	if err != nil {
		return err
	}
	p := pprof.Lookup("mutex")
	return p.WriteTo(w, 0)
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

func getVersion(zipWriter *zip.Writer) error {
	w, err := zipWriter.Create("versions.json")
	if err != nil {
		return err
	}
	versions := version.GetAllVersions()
	data, err := json.Marshal(versions)
	if err != nil {
		return err
	}

	_, err = w.Write(data)
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

		if err := getVersion(zipWriter); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := getPrometheusMetrics(zipWriter, "metrics-1"); err != nil {
			log.Error(err)
		}

		if err := getMemory(zipWriter); err != nil {
			log.Error(err)
		}

		if err := getGoroutines(zipWriter); err != nil {
			log.Error(err)
		}

		if err := getBlock(zipWriter); err != nil {
			log.Error(err)
		}

		if err := getMutex(zipWriter); err != nil {
			log.Error(err)
		}

		if err := getCPU(zipWriter, cpuProfileDuration); err != nil {
			log.Error(err)
		}

		if err := getPrometheusMetrics(zipWriter, "metrics-2"); err != nil {
			log.Error(err)
		}

		if withLogs {
			if err := getLogs(zipWriter); err != nil {
				log.Error(err)
			}
		}
	}
}
