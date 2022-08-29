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

package dashboard

import (
	"bytes"
	_ "embed" // nolint
	"fmt"
	"html/template"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethinx/xk6-dashboard/internal"
	"github.com/gorilla/schema"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/metrics"
	"go.k6.io/k6/output"
)

// Register the extensions on module initialization.
func init() {
	output.RegisterExtension("dashboard", New)
}

const (
	pathMetrics   = "/api/metrics"
	defaultPort   = 5665
	defaultPeriod = 10
)

type options struct {
	Port   int
	Host   string
	Period int
	UI     string
	Wait   int
}

type Output struct {
	output.SampleBuffer

	*internal.PrometheusAdapter

	flusher        *output.PeriodicFlusher
	addr           string
	arg            string
	logger         logrus.FieldLogger
	sampleChannel  chan []metrics.SampleContainer
	wg             sync.WaitGroup
	workGroupCount int64
}

func New(params output.Params) (output.Output, error) {
	registry := prometheus.NewRegistry()
	o := &Output{
		PrometheusAdapter: internal.NewPrometheusAdapter(registry, params.Logger, "", ""),
		arg:               params.ConfigArgument,
		logger:            params.Logger,
		flusher:           nil,
		addr:              "",
		sampleChannel:     make(chan []metrics.SampleContainer, 10),
		workGroupCount:    0,
	}

	return o, nil
}

func (o *Output) Description() string {
	return fmt.Sprintf("dashboard (%s)", o.addr)
}

func getopts(qs string) (*options, error) {
	opts := &options{
		Port:   defaultPort,
		Host:   "",
		Period: defaultPeriod,
	}

	if qs == "" {
		return opts, nil
	}

	v, err := url.ParseQuery(qs)
	if err != nil {
		return nil, err
	}

	decoder := schema.NewDecoder()

	if err = decoder.Decode(opts, v); err != nil {
		return nil, err
	}

	return opts, nil
}

func (o *Output) handler(opts *options) (http.Handler, error) {
	tmpl, err := template.New("index.html").Parse(index)
	if err != nil {
		return nil, err
	}

	mux := http.DefaultServeMux
	mux.Handle(pathMetrics, o.PrometheusAdapter.Handler())

	u, err := url.Parse(opts.UI)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "" {
		u.Scheme = "https"
	}

	var buff bytes.Buffer

	err = tmpl.Execute(&buff, map[string]string{"ui": u.String()})
	if err != nil {
		return nil, err
	}

	page := buff.Bytes()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)

			return
		}

		w.Write(page) // nolint:errcheck
	})

	return mux, nil
}

func (o *Output) Start() error {
	opts, err := getopts(o.arg)
	if err != nil {
		return err
	}

	o.addr = fmt.Sprintf("%s:%d", opts.Host, opts.Port)

	listener, err := net.Listen("tcp", o.addr)
	if err != nil {
		return err
	}

	handler, err := o.handler(opts)
	if err != nil {
		return err
	}

	go func() {
		if err := http.Serve(listener, handler); err != nil {
			o.logger.Error(err)
		}
	}()

	go o.MetricWorker()

	o.flusher, err = output.NewPeriodicFlusher(time.Duration(opts.Period)*time.Second, o.flushMetrics)
	if err != nil {
		return err
	}

	return nil
}

func (o *Output) MetricWorker() {
	for {
		select {
		case sampleGroup := <-o.sampleChannel:
			go func(*Output) {
				defer o.wg.Done()
				defer func() {
					atomic.AddInt64(&o.workGroupCount, -1)
				}()
				o.wg.Add(1)
				atomic.AddInt64(&o.workGroupCount, 1)

				for _, sc := range sampleGroup {
					samples := sc.GetSamples()

					for _, entry := range samples {
						o.HandleSample(&entry)
					}

				}

			}(o)
		}
	}
}

func (o *Output) flushMetrics() {
	bufferSamples := o.GetBufferedSamples()
	lower := 0
	upper := 0
	batchSize := 10000
	bufferSize := len(bufferSamples)

	if batchSize > bufferSize {
		upper = bufferSize
	} else {
		upper = batchSize
	}

	for lower <= bufferSize-1 {
		samplesGroup := bufferSamples[lower:upper]
		o.sampleChannel <- samplesGroup

		lower = upper
		upper += batchSize

		if upper >= bufferSize {
			upper = bufferSize
		}

	}
	o.logger.WithField("Count", o.workGroupCount).Info("Work Group")
}

func (o *Output) Stop() error {
	defer close(o.sampleChannel)

	o.flusher.Stop()
	o.wg.Wait()

	opts, err := getopts(o.arg)
	if err != nil {
		return err
	}

	if opts.Wait > 0 {
		o.logger.Infof("All set and wait %ds", opts.Wait)
		time.Sleep(time.Duration(opts.Wait) * time.Second)
	}

	return nil
}

//go:embed index.html
var index string
