// MIT License
//
// Copyright (c) 2021 Iván Szkiba
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package internal

import (
	// "fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/metrics"
)

type PrometheusAdapter struct {
	// metrics   map[string]interface{}
	Subsystem      string
	Namespace      string
	logger         logrus.FieldLogger
	metrics        sync.Map
	registry       *prometheus.Registry
	lock           sync.RWMutex
	builtinMetrics builtinMetrics
}

var builtinMetricsMap = map[string]string{
	"vus":                "Current number of active virtual users",
	"vus_max":            "Max possible number of virtual users",
	"iterations":         "The aggregate number of times the VUs in the test have executed",
	"iteration_duration": "The time it took to complete one full iteration",
	"dropped_iterations": "The number of iterations that could not be started",
	"data_received":      "The amount of received data",
	"data_sent":          "The amount of data sent",
	"checks":             "The rate of successful checks",

	"http_reqs":                "How many HTTP requests has k6 generated, in total",
	"http_req_blocked":         "Time spent blocked  before initiating the request",
	"http_req_connecting":      "Time spent establishing TCP connection",
	"http_req_tls_handshaking": "Time spent handshaking TLS session",
	"http_req_sending":         "Time spent sending data",
	"http_req_waiting":         "Time spent waiting for response",
	"http_req_receiving":       "Time spent receiving response data",
	"http_req_duration":        "Total time for the request",
	"http_req_failed":          "The rate of failed requests",
}

type builtinMetrics struct {
	VUS                          prometheus.Gauge
	VUSMax                       prometheus.Gauge
	HTTPReqBlockedCurrent        prometheus.Gauge
	HTTPReqConnectingCurrent     prometheus.Gauge
	HTTPReqDurationCurrent       prometheus.Gauge
	HTTPReqFailedCurrent         prometheus.Gauge
	HTTPReqReceivingCurrent      prometheus.Gauge
	HTTPReqSendingCurrent        prometheus.Gauge
	HTTPReqTLSHandshakingCurrent prometheus.Gauge
	HTTPReqWaitingCurrent        prometheus.Gauge
	IterationDurationCurrent     prometheus.Gauge
	DataReceived                 prometheus.Counter
	DataSent                     prometheus.Counter
	HTTPReqs                     prometheus.Counter
	Iterations                   prometheus.Counter
	DroppedIterations            prometheus.Counter
	HTTPReqBlocked               prometheus.Histogram
	HTTPReqConnecting            prometheus.Histogram
	HTTPReqDuration              prometheus.Histogram
	HTTPReqReceiving             prometheus.Histogram
	HTTPReqSending               prometheus.Histogram
	HTTPReqTLSHandshaking        prometheus.Histogram
	HTTPReqWaiting               prometheus.Histogram
	IterationDuration            prometheus.Histogram
	Checks                       prometheus.Histogram
	HTTPReqFailed                prometheus.Histogram
}

type Counter struct {
	Counter prometheus.Counter
	Help    string
}

func NewCounter(registry *prometheus.Registry, namespace, subsystem, name, help string) prometheus.Counter {
	metric := prometheus.NewCounter(prometheus.CounterOpts{ // nolint:exhaustivestruct
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	})

	return metric
}

type Gauge struct {
	Gauge prometheus.Gauge
	Help  string
}

func NewGauge(registry *prometheus.Registry, namespace, subsystem, name, help string) prometheus.Gauge {
	metric := prometheus.NewGauge(prometheus.GaugeOpts{ // nolint:exhaustivestruct
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	})

	return metric
}

type Summary struct {
	Summary prometheus.Summary
	Help    string
}

func NewSummary(registry *prometheus.Registry, namespace, subsystem, name, help string) prometheus.Summary {
	metric := prometheus.NewSummary(prometheus.SummaryOpts{ // nolint:exhaustivestruct
		Namespace:  namespace,
		Subsystem:  subsystem,
		Name:       name,
		Help:       help,
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.01, 0.99: 0.001, 1: 0}, // nolint:gomnd
	})

	return metric
}

type Histogram struct {
	Histogram prometheus.Histogram
	Help      string
}

func NewHistogram(registry *prometheus.Registry, namespace, subsystem, name, help string, buckets []float64) prometheus.Histogram {
	if len(buckets) == 0 {
		buckets = append([]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9}, prometheus.ExponentialBuckets(1, 2, 16)...)
	}
	metric := prometheus.NewHistogram(prometheus.HistogramOpts{ // nolint:exhaustivestruct
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	})

	return metric
}

func NewPrometheusAdapter(registry *prometheus.Registry, logger logrus.FieldLogger, ns, sub string) *PrometheusAdapter {
	builtinMetrics := builtinMetrics{
		VUS:                          NewGauge(registry, ns, sub, "vus", "Current number of active virtual users"),
		VUSMax:                       NewGauge(registry, ns, sub, "vus_max", "Max possible number of virtual users"),
		HTTPReqBlockedCurrent:        NewGauge(registry, ns, sub, "http_req_blocked_current", "Time spent blocked  before initiating the request"),
		HTTPReqConnectingCurrent:     NewGauge(registry, ns, sub, "http_req_connecting_current", "Time spent establishing TCP connection"),
		HTTPReqDurationCurrent:       NewGauge(registry, ns, sub, "http_req_duration_current", "Total time for the request"),
		HTTPReqReceivingCurrent:      NewGauge(registry, ns, sub, "http_req_receiving_current", "Time spent receiving response data"),
		HTTPReqSendingCurrent:        NewGauge(registry, ns, sub, "http_req_sending_current", "Time spent sending data"),
		HTTPReqTLSHandshakingCurrent: NewGauge(registry, ns, sub, "http_req_tls_handshaking_current", "Time spent handshaking TLS session"),
		HTTPReqWaitingCurrent:        NewGauge(registry, ns, sub, "http_req_waiting_current", "Time spent waiting for response"),
		IterationDurationCurrent:     NewGauge(registry, ns, sub, "iteration_duration_current", "The time it took to complete one full iteration"),
		DataReceived:                 NewCounter(registry, ns, sub, "data_received", "The amount of received data"),
		DataSent:                     NewCounter(registry, ns, sub, "data_sent", "The amount of data sent"),
		HTTPReqs:                     NewCounter(registry, ns, sub, "http_reqs", "How many HTTP requests has k6 generated, in total"),
		Iterations:                   NewCounter(registry, ns, sub, "iterations", "The aggregate number of times the VUs in the test have executed"),
		DroppedIterations:            NewCounter(registry, ns, sub, "dropped_iterations", "The number of iterations that could not be started"),
		HTTPReqBlocked:               NewHistogram(registry, ns, sub, "http_req_blocked", "time spent blocked  before initiating the request", []float64{}),
		HTTPReqConnecting:            NewHistogram(registry, ns, sub, "http_req_connecting", "time spent establishing tcp connection", []float64{}),
		HTTPReqReceiving:             NewHistogram(registry, ns, sub, "http_req_receiving", "time spent receiving response data", []float64{}),
		HTTPReqSending:               NewHistogram(registry, ns, sub, "http_req_sending", "time spent sending data", []float64{}),
		HTTPReqTLSHandshaking:        NewHistogram(registry, ns, sub, "http_req_tls_handshaking", "time spent handshaking tls session", []float64{}),
		HTTPReqWaiting:               NewHistogram(registry, ns, sub, "http_req_waiting", "time spent waiting for response", []float64{}),
		HTTPReqDuration:              NewHistogram(registry, ns, sub, "http_req_duration", "total time for the request", []float64{}),
		IterationDuration:            NewHistogram(registry, ns, sub, "iteration_duration", "the time it took to complete one full iteration", []float64{}),
		Checks:                       NewHistogram(registry, ns, sub, "checks", "The rate of successful checks", []float64{}),
		HTTPReqFailed:                NewHistogram(registry, ns, sub, "http_req_failed", "The rate of failed requests", []float64{}),
	}

	// register builtin metrics
	metrics := []prometheus.Collector{
		builtinMetrics.VUS,
		builtinMetrics.VUSMax,
		builtinMetrics.HTTPReqBlockedCurrent,
		builtinMetrics.HTTPReqConnectingCurrent,
		builtinMetrics.HTTPReqDurationCurrent,
		builtinMetrics.HTTPReqReceivingCurrent,
		builtinMetrics.HTTPReqSendingCurrent,
		builtinMetrics.HTTPReqTLSHandshakingCurrent,
		builtinMetrics.HTTPReqWaitingCurrent,
		builtinMetrics.IterationDurationCurrent,
		builtinMetrics.DataReceived,
		builtinMetrics.DataSent,
		builtinMetrics.HTTPReqs,
		builtinMetrics.Iterations,
		builtinMetrics.DroppedIterations,
		builtinMetrics.HTTPReqBlocked,
		builtinMetrics.HTTPReqConnecting,
		builtinMetrics.HTTPReqReceiving,
		builtinMetrics.HTTPReqSending,
		builtinMetrics.HTTPReqTLSHandshaking,
		builtinMetrics.HTTPReqWaiting,
		builtinMetrics.HTTPReqDuration,
		builtinMetrics.Checks,
		builtinMetrics.HTTPReqFailed,
	}

	for _, collector := range metrics {
		if err := registry.Register(collector); err != nil {
			return nil
		}
	}

	return &PrometheusAdapter{
		Subsystem:      sub,
		Namespace:      ns,
		logger:         logger,
		registry:       registry,
		builtinMetrics: builtinMetrics,
	}
}

func (a *PrometheusAdapter) Handler() http.Handler {
	return promhttp.HandlerFor(a.registry, promhttp.HandlerOpts{}) // nolint:exhaustivestruct
}

func (a *PrometheusAdapter) HandleSample(sample *metrics.Sample) {
	var handler func(*metrics.Sample)

	switch sample.Metric.Type {
	case metrics.Counter:
		handler = a.handleCounter
	case metrics.Gauge:
		handler = a.handleGauge
	case metrics.Rate:
		handler = a.handleRate
	case metrics.Trend:
		handler = a.handleTrend
	default:
		a.logger.Warnf("Unknown metric type: %v", sample.Metric.Type)

		return
	}

	handler(sample)
}

func (a *PrometheusAdapter) handleCounter(sample *metrics.Sample) {
	switch sample.Metric.Name {
	case "data_received":
		a.builtinMetrics.DataReceived.Add(sample.Value)
	case "data_sent":
		a.builtinMetrics.DataSent.Add(sample.Value)
	case "http_reqs":
		a.builtinMetrics.HTTPReqs.Add(sample.Value)
	case "iterations":
		a.builtinMetrics.Iterations.Add(sample.Value)
	default:
		return
	}
}

func (a *PrometheusAdapter) handleGauge(sample *metrics.Sample) {
	switch sample.Metric.Name {
	case "vus":
		a.builtinMetrics.VUS.Set(sample.Value)
	case "vus_max":
		a.builtinMetrics.VUSMax.Set(sample.Value)
	case "http_req_blocked_current":
		a.builtinMetrics.HTTPReqBlockedCurrent.Set(sample.Value)
	case "http_req_connecting_current":
		a.builtinMetrics.HTTPReqConnectingCurrent.Set(sample.Value)
	case "http_req_duration_current":
		a.builtinMetrics.HTTPReqDurationCurrent.Set(sample.Value)
	case "http_req_receiving_current":
		a.builtinMetrics.HTTPReqReceivingCurrent.Set(sample.Value)
	case "http_req_sending_current":
		a.builtinMetrics.HTTPReqSendingCurrent.Set(sample.Value)
	case "http_req_tls_handshaking_current":
		a.builtinMetrics.HTTPReqTLSHandshakingCurrent.Set(sample.Value)
	case "http_req_waiting_current":
		a.builtinMetrics.HTTPReqWaitingCurrent.Set(sample.Value)
	case "iteration_duration_current":
		a.builtinMetrics.IterationDurationCurrent.Set(sample.Value)
	default:
		return
	}
}

func (a *PrometheusAdapter) handleRate(sample *metrics.Sample) {
	switch sample.Metric.Name {
	case "checks":
		a.builtinMetrics.Checks.Observe(sample.Value)
	case "http_req_failed":
		a.builtinMetrics.HTTPReqFailed.Observe(sample.Value)
	default:
		return
	}
}

func (a *PrometheusAdapter) handleTrend(sample *metrics.Sample) {
	switch sample.Metric.Name {
	case "http_req_blocked":
		a.builtinMetrics.HTTPReqBlockedCurrent.Set(sample.Value)
		a.builtinMetrics.HTTPReqBlocked.Observe(sample.Value)
	case "http_req_connecting":
		a.builtinMetrics.HTTPReqConnectingCurrent.Set(sample.Value)
		a.builtinMetrics.HTTPReqConnecting.Observe(sample.Value)
	case "http_req_duration":
		a.builtinMetrics.HTTPReqDurationCurrent.Set(sample.Value)
		a.builtinMetrics.HTTPReqDuration.Observe(sample.Value)
	case "http_req_receiving":
		a.builtinMetrics.HTTPReqReceivingCurrent.Set(sample.Value)
		a.builtinMetrics.HTTPReqReceiving.Observe(sample.Value)
	case "http_req_sending":
		a.builtinMetrics.HTTPReqSendingCurrent.Set(sample.Value)
		a.builtinMetrics.HTTPReqSending.Observe(sample.Value)
	case "http_req_tls_handshaking":
		a.builtinMetrics.HTTPReqTLSHandshakingCurrent.Set(sample.Value)
		a.builtinMetrics.HTTPReqTLSHandshaking.Observe(sample.Value)
	case "http_req_waiting":
		a.builtinMetrics.HTTPReqWaitingCurrent.Set(sample.Value)
		a.builtinMetrics.HTTPReqWaiting.Observe(sample.Value)
	case "iteration_duration":
		a.builtinMetrics.IterationDurationCurrent.Set(sample.Value)
		a.builtinMetrics.IterationDuration.Observe(sample.Value)
	default:
		return
	}
}
