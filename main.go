package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"embed"

	log "github.com/sirupsen/logrus"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/api"
	PromAPIV1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// Prometheous metric type and description
type metricDetails struct {
	Type        string
	Description string
}

type config struct {
	SERVE_PORT       int    `envconfig:"SERVE_PORT" default:"8080""`
	REFRESH_INTERVAL int    `envconfig:"REFRESH_INTERVAL" default:"14400""`
	PROMETHEUS_URL   string `envconfig:"PROMETHEUS_URL" required:"true""`
}

var metrics = make(map[string]metricDetails)
var targets = make(map[string]bool)

//go:embed static
var embededFiles embed.FS

var appConfig config

func init() {
	err := envconfig.Process("", &appConfig)
	if err != nil {
		log.Fatal("application startup error: \n", err)
	}
}

func main() {
	client, err := api.NewClient(api.Config{
		Address: appConfig.PROMETHEUS_URL,
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}
	log.Debug(fmt.Sprint("Connecting to prometheus URL:", appConfig.PROMETHEUS_URL))

	v1api := PromAPIV1.NewAPI(client)
	log.Debug("Extracting metrics with their descriptions.")

	err = prepareDate(v1api)
	if err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(time.Duration(appConfig.REFRESH_INTERVAL) * time.Second)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				log.Info("Refreshing data.")
				err = prepareDate(v1api)
				if err != nil {
					log.Error("Failed to update date ", err)
				}
			}
		}
	}()

	http.Handle("/", http.FileServer(getFileSystem(false)))
	http.HandleFunc("/targets", getTargets)
	http.HandleFunc("/metrics", getMetrics)
	log.Print(fmt.Sprint("App is running on port:", appConfig.SERVE_PORT))
	err = http.ListenAndServe(fmt.Sprint(":", appConfig.SERVE_PORT), nil)
	if err != nil {
		log.Fatal("error starting up app", err)
	}
}

// Queries prometheus for configured targets
// pulls the metrics from each target
// extracts type and description for each metric
func prepareDate(v1api PromAPIV1.API) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("Retrieving prometheus targets")
	res, err := v1api.Targets(ctx)
	if err != nil {
		log.Error("Error getting targets from Prometheus: %v\n", err)
		return err
	}

	for _, v := range res.Active {
		target := v.ScrapeURL
		resp, err := http.Get(target)
		if err != nil {
			log.Warn("Target: " + target + " is unreachable")
			targets[target] = false
			continue
		}
		targets[target] = true
		defer resp.Body.Close()

		log.Info("Retrieving metrics details from ", target)
		body, err := ioutil.ReadAll(resp.Body)
		r := strings.NewReader(string(body))

		// loop over each line
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			// select records which are comments
			// format is typically
			// # HELP metric_name metric_description.
			// # TYPE metric_name metric_type
			if strings.HasPrefix(line, "#") {
				stringSlice := strings.Split(line, " ")
				var m metricDetails
				key, metric, value := stringSlice[1], stringSlice[2], stringSlice[3:]

				if _, ok := metrics[metric]; ok {
					m = metrics[metric]
				}

				if key == "HELP" {
					m.Description = strings.Join(value, " ")
				}

				if key == "TYPE" {
					m.Type = value[0]
				}

				metrics[metric] = m
			}
		}
		if err := scanner.Err(); err != nil {
			log.Error(os.Stderr, "reading standard input:", err)
			return err
		}
	}
	return nil
}

// Prints prometheous Targets as JSON string
func getTargets(w http.ResponseWriter, r *http.Request) {
	t, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		log.Errorf("Panic: %s", err)
	}
	fmt.Fprintf(w, string(t))
}

// Prints prometheus Metrics details as JSON string
func getMetrics(w http.ResponseWriter, r *http.Request) {
	m, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		log.Errorf("Panic: %s", err)
	}
	fmt.Fprintf(w, string(m))
}

// embeds filesystem with html file
func getFileSystem(useOS bool) http.FileSystem {
	f := "static"
	if useOS {
		log.Print("OS filesystem")
		return http.FS(os.DirFS(f))
	}

	log.Print("using embed mode")
	fsys, err := fs.Sub(embededFiles, f)
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}
