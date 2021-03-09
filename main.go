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
var targets = make(map[string]bool)

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

	http.HandleFunc("/", index)
	http.HandleFunc("/targets", getTargets)
	http.HandleFunc("/metrics", getMetrics)
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
			targets[target] = false
			continue
		}
		targets[target] = true
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
}

func getTargets(w http.ResponseWriter, r *http.Request) {
	t, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		log.Errorf("Panic: %s", err)
	}
	fmt.Fprintf(w, string(t))
}

func getMetrics(w http.ResponseWriter, r *http.Request) {
	m, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		log.Errorf("Panic: %s", err)
	}
	fmt.Fprintf(w, string(m))
}

func index(w http.ResponseWriter, r *http.Request) {
	str := `
	<html lang="en"> 

	<head> 
		<meta charset="UTF-8"> 
		<title>Prometheus Metrics Dictionary</title> 
		<script src= "https://code.jquery.com/jquery-3.5.1.js"></script> 
		<script src="https://cdn.jsdelivr.net/npm/bootstrap@5.0.0-beta2/dist/js/bootstrap.bundle.min.js" integrity="sha384-b5kHyXgcpbZJO/tY9Ul7kGkf1S0CWuKcCD38l8YkeH8z8QjE0GmW1gYU5S9FOnJ0" crossorigin="anonymous"></script>
		<link href="https://cdn.jsdelivr.net/npm/bootstrap@5.0.0-beta2/dist/css/bootstrap.min.css" rel="stylesheet" integrity="sha384-BmbxuPwQa2lc/FVzBcNJ7UAyJxM6wuqIj61tLrc4wSX0szH/Ev+nYRRuWlolflfl" crossorigin="anonymous">
	</head> 
	
	<body> 
		<div class="container"> 
			<h1 align="center">Prometheus Metrics Dictionary</h1> 
			<div class="mb-3">
				<label for="myInput" class="form-label">Search</label>
				<input id="myInput"  type="text" class="form-control"  placeholder="search for metric">
			</div>
	
			<div class="alert alert-info alert-dismissible fade show" role="alert">
				<strong>Targets</strong>
				<ul id="targets"></ul>
				<button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
			</div>
			<!-- TABLE CONSTRUCTION-->
			<table id='table' class="table table-striped table-hover"> 
				<!-- HEADING FORMATION -->
				<tr> 
					<th>Name</th> 
					<th>Type</th> 
					<th>Description</th> 
				</tr> 
	
				<script>
					$(document).ready(function () { 
						// Data Filter
						$("#myInput").on("keyup", function() {
							var value = $(this).val().toLowerCase();
							$("#table tr").filter(function() {
							$(this).toggle($(this).text().toLowerCase().indexOf(value) > -1)
							});
						});
	
						// load targets information
						$.getJSON("/targets", 
							function (data) { 
								var targets = ''; 
								// ITERATING THROUGH OBJECTS 
								$.each(data, function (key, value) { 
									console.log(key , value);
									// //CONSTRUCTION OF ROWS HAVING 
									// // DATA FROM JSON OBJECT 
									var status = " (OK) "
									if (value == false) {
										var status = " (unreachable) "
									} 
	
									targets += '<li>'; 
									targets += key + status;   
									targets += '</li>'; 
							}); 
								
							//INSERTING ROWS INTO TABLE 
							$('#targets').append(targets); 
						}); 
						
						// FETCHING DATA FROM JSON FILE 
						$.getJSON("/metrics", 
								function (data) { 
									var metrics = ''; 
	
									// ITERATING THROUGH OBJECTS 
									$.each(data, function (key, value) { 
										console.log(key , value)
										// //CONSTRUCTION OF ROWS HAVING 
										// // DATA FROM JSON OBJECT 
										metrics += '<tr>'; 
										metrics += '<td>' + key + '</td>'; 
										metrics += '<td>' + value.Type + '</td>'; 
										metrics += '<td>' + value.Description + '</td>'; 
										metrics += '</tr>'; 
									}); 
									
									//INSERTING ROWS INTO TABLE 
									$('#table').append(metrics); 
						}); 
					}); 
				</script> 
		</div> 
	</body> 
	
	</html> 
	`
	fmt.Fprintf(w, str)
}
