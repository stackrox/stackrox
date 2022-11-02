# go-promhttp

go-promhttp provides wrappers around the http package objects for monitoring with Prometheus, namely the http.Client, and http.Handler.

### Requirements

minimum golang version of `1.9`

## Wrapping the http.Client
For example, you can instrument your http requests sent using the promhttp.Client as follows:

``` go
httpClient := &promhttp.Client{
	Client: http.DefaultClient,
	Registerer: prometheus.DefaultRegister,
}
githubClient, _ := httpClient.ForRecipient("github")

resp, err := githubClient.Get("https://api.github.com/repos/travelaudience/go-promhttp/issues")
...
```

Doing so will give you prometheus metrics such as:

| metric                             | description                               |
|------------------------------------|-------------------------------------------|
| requests_total                     | A counter for outgoing requests.          |
| request_duration_histogram_seconds | A histogram of outgoing request latencies. |
| dns_duration_histogram_seconds     | Trace dns latency histogram.              |
| tls_duration_histogram_seconds     | Trace tls latency histogram.              |
| in_flight_requests                 | A gauge of in-flight outgoing requests.    |

By calling httpClient.ForRecipient("github"), all of these metrics will be tagged with the label `"recipient": "github"`

## Wrapping the http.Handler
Simmilarly, a http.Handler can be instrumented for monitoring via the promhttp.ServeMux. By running the following code,

``` go
mux := &promhttp.ServeMux {
	ServeMux: &http.ServeMux{}
}

mux.Handle("/issues", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	...
}))
```

we have the mux handle incoming requests and generating metrics. The metrics generated are as follows:

| metric                             | description                                              |
|------------------------------------|----------------------------------------------------------|
| request_duration_histogram_seconds | The request time duration .                              |
| requests_total                     | The total number of requests received.                   |
| request_size_histogram_bytes       | The request size in bytes.                               |
| response_size_histogram_bytes      | Thes response size in bytes.                             |
| in_flight_requests                 | The number of http requests which are currently running. |

And all of these metrics in our example would have the label `"path": "/issues"`.


## Contributing

Contributions are welcomed! Read the [Contributing Guide](.github/CONTRIBUTING.md) for more information.

## Licensing

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details
