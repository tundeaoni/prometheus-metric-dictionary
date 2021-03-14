build-image:
	docker build -t tundeaoni/prometheus-metrics-dictionary  .
	docker push tundeaoni/prometheus-metrics-dictionary