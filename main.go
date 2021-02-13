package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/api"
	PromAPIV1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type metricDetails struct {
	Type        string
	Description string
}

type config struct {
	SERVE_PORT     int    `envconfig:"SERVE_PORT" default:"8080""`
	PROMETHEUS_URL string `envconfig:"PROMETHEUS_URL" required:"true""`
}

const ASSETS_DIR = "./assets/"

var metrics = make(map[string]metricDetails)

var appConfig config

func init() {
	err := envconfig.Process("", &appConfig)
	if err != nil {
		log.Fatal("application startup error", err)
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

	v1api := PromAPIV1.NewAPI(client)
	prepareDate(v1api)

	http.Handle("/", http.FileServer(http.Dir(ASSETS_DIR)))
	log.Print(fmt.Sprint("App is running on port:", appConfig.SERVE_PORT))
	err = http.ListenAndServe(fmt.Sprint(":", appConfig.SERVE_PORT), nil)
	if err != nil {
		log.Fatal("error starting up app", err)
	}
}

func prepareDate(v1api PromAPIV1.API) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := v1api.Targets(ctx)
	if err != nil {
		fmt.Printf("Error getting Targets from Prometheus: %v\n", err)
		os.Exit(1)
	}

	for _, v := range res.Active {
		target := v.ScrapeURL
		resp, err := http.Get(target)
		if err != nil {
			log.Warn("Target: " + target + " is unreachable")
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		r := strings.NewReader(string(body))

		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
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
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
	}
	jsonString, err := json.MarshalIndent(metrics, "", "  ")
	err = ioutil.WriteFile(ASSETS_DIR+".data", jsonString, 0644)
	if err != nil {
		log.Error(err)
	}
}
