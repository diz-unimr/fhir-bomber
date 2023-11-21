package client

import (
	"encoding/json"
	"fhir-bomber/pkg/config"
	"fhir-bomber/pkg/monitoring"
	"github.com/opsway-io/go-httpstat"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Bomber struct {
	Config   config.AppConfig
	Requests []Request
	Workers  int
	Metrics  *monitoring.Metrics
}

type Request struct {
	Name string
	Url  string
}

type TraceResult struct {
	ResponseCode int
	Total        time.Duration
	Name         string
	Url          string
}

func NewBomber(config config.AppConfig, m *monitoring.Metrics) *Bomber {
	return &Bomber{
		Config:   config,
		Requests: loadRequests(config.Bomber.Requests),
		Metrics:  m,
		Workers:  config.Bomber.Workers,
	}
}

func loadRequests(f string) []Request {
	file, err := os.Open(f)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open requests file")
		os.Exit(1)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	bytes, _ := io.ReadAll(file)
	var requests []Request

	err = json.Unmarshal(bytes, &requests)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to parse json requests file: %s", file.Name())
		os.Exit(1)
	}

	return requests
}

func (b *Bomber) Run() {

	numJobs := len(b.Requests)
	var wg sync.WaitGroup

	for run := 1; ; run++ {

		jobs := make(chan Request, numJobs)
		// workers
		b.createWorkers(&wg, jobs, b.Workers)

		log.Info().Int("iteration", run).Msg("Run started")

		// jobs
		for i, j := range b.Requests {
			if len(strings.TrimSpace(j.Name)) == 0 {
				j.Name = strconv.Itoa(i + 1)
			}
			jobs <- j
		}
		close(jobs)

		wg.Wait()
		log.Info().Int("run", run).Msgf("Run [%d] done", run)
		time.Sleep(b.Config.Bomber.Interval)

	}
}

func (b *Bomber) createWorkers(wg *sync.WaitGroup, jobs <-chan Request, count int) {
	// worker
	for w := 1; w <= count; w++ {

		wg.Add(1)
		worker := w
		log.Info().Int("worker", worker).Msg("Worker created")

		go func(jobs <-chan Request) {
			defer wg.Done()

			for j := range jobs {
				log.Info().Str("job", j.Name).Int("worker", worker).Msg("New job")
				b.execute(j)
				log.Info().Str("job", j.Name).Int("worker", worker).Msg("Job done")
			}
		}(jobs)
	}
}

func (b *Bomber) execute(req Request) {
	target, err := url.JoinPath(b.Config.Fhir.Base, req.Url)
	if err != nil {
		log.Error().Err(err).Msg("Building request path failed")
		return
	}

	result, err := b.executeRequest(target)
	if err != nil {
		log.Error().Err(err).Msg("Failed to execute request")
		return
	}

	log.Info().Str("request", req.Name).Int("code", result.ResponseCode).Dur("latency (ms)", result.Total).Msg("Request completed")
	// metrics
	b.Metrics.RequestDuration.With(prometheus.Labels{"name": req.Name, "code": strconv.Itoa(result.ResponseCode)}).Observe(result.Total.Seconds())

}

func (b *Bomber) executeRequest(url string) (*TraceResult, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create GET request")
		return nil, err
	}
	// trace
	var result httpstat.Result
	ctx := httpstat.WithHTTPStat(req.Context(), &result)
	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", "application/fhir+json")
	if b.Config.Fhir.Auth != nil {
		req.SetBasicAuth(b.Config.Fhir.Auth.User, b.Config.Fhir.Auth.Password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return nil, err
	}
	now := time.Now()
	_ = resp.Body.Close()
	result.End(now)

	return &TraceResult{
		Url:          url,
		ResponseCode: resp.StatusCode,
		Total:        result.Total,
	}, nil
}
