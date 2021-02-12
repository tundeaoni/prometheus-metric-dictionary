package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"net/http"
	"io/ioutil"
	"strings"
	"bufio"
	"encoding/json"
	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
)

type metricDetails struct {
	Type string
	Description string
}

var metrics = make( map[string]metricDetails)

var prom_url = "http://demo.robustperception.io:9090"

func main() {
	client, err := api.NewClient(api.Config{
		Address: prom_url,
	})
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	v1api := v1.NewAPI(client)
	prepareDate(v1api)

	http.HandleFunc("/", index)
	log.Info("Serving on :8080...")
    http.ListenAndServe(":8080", nil)

}

func index(w http.ResponseWriter, r *http.Request){
	data, err := ioutil.ReadFile(".data")
	if err != nil {
		log.Errorf("Panic: %s", err)
	}
    fmt.Fprintf(w, string(data))
}


func prepareDate(v1api v1.API) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := v1api.Targets(ctx)
	if err != nil {
		fmt.Printf("Error getting Targets from Prometheus: %v\n", err)
		os.Exit(1)
	}

	for _,v := range res.Active {
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
				key , metric , value := stringSlice[1], stringSlice[2], stringSlice[3:]

				if _, ok := metrics[metric]; ok {
					m = metrics[metric]
				}

				if key == "HELP"{
					m.Description = strings.Join(value, " ")
				}

				if key == "TYPE"{
					m.Type = value[0]
				}

				metrics[metric] = m
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}
	}
	jsonString, err := json.MarshalIndent(metrics ,"", "  ")
	err = ioutil.WriteFile(".data", jsonString, 0644)
	if err != nil {
		fmt.Println(err)
	}
	// return string(jsonString)
}
