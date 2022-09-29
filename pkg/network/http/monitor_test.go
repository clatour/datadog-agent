// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux_bpf
// +build linux_bpf

package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/DataDog/datadog-agent/pkg/network/tracer"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	nethttp "net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/DataDog/datadog-agent/pkg/network/config"
	"github.com/DataDog/datadog-agent/pkg/network/http/testutil"
	netlink "github.com/DataDog/datadog-agent/pkg/network/netlink/testutil"
	"github.com/DataDog/datadog-agent/pkg/util/kernel"
)

const (
	kb = 1024
	mb = 1024 * kb
)

var (
	emptyBody = []byte(nil)
)

var (
	disableTLSVerification = sync.Once{}
)

func skipTestIfKernelNotSupported(t *testing.T) {
	currKernelVersion, err := kernel.HostVersion()
	require.NoError(t, err)
	if currKernelVersion < MinimumKernelVersion {
		t.Skip(fmt.Sprintf("HTTP feature not available on pre %s kernels", MinimumKernelVersion.String()))
	}
}

func writeTempFile(pattern string, content string) (*os.File, error) {
	f, err := ioutil.TempFile("", pattern)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return nil, err
	}

	return f, nil
}

func rawConnect(ctx context.Context, t *testing.T, host string, port string) {
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("failed connecting to port %s:%s", host, port)
		default:
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Second)
			if err != nil {
				continue
			}
			if conn != nil {
				conn.Close()
				return
			}
		}
	}

}

const pythonSSLServerFormat = `import http.server, ssl

class RequestHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        status_code = int(self.path.split("/")[1])
        self.send_response(status_code)
        self.end_headers()
        self.wfile.write(b'Hello, world!')

server_address = ('127.0.0.1', 8001)
httpd = http.server.HTTPServer(server_address, RequestHandler)
httpd.socket = ssl.wrap_socket(httpd.socket,
                               server_side=True,
                               certfile='%s',
                               keyfile='%s',
                               ssl_version=ssl.PROTOCOL_TLS)
httpd.serve_forever()
`

func TestOpenSSLVersions(t *testing.T) {
	skipTestIfKernelNotSupported(t)

	disableTLSVerification.Do(func() {
		nethttp.DefaultTransport.(*nethttp.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	})
	curDir, _ := testutil.CurDir()
	crtPath := filepath.Join(curDir, "testdata/cert.pem.0")
	keyPath := filepath.Join(curDir, "testdata/server.key")
	pythonSSLServer := fmt.Sprintf(pythonSSLServerFormat, crtPath, keyPath)
	scriptFile, err := writeTempFile("python_openssl_script", pythonSSLServer)
	require.NoError(t, err)
	defer scriptFile.Close()

	cmd := exec.Command("python3", scriptFile.Name())
	go func() {
		err := cmd.Start()
		if err != nil {
			fmt.Println(err)
		}
	}()
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	portCtx, cancelPortCtx := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
	rawConnect(portCtx, t, "127.0.0.1", "8001")
	cancelPortCtx()

	cfg := config.New()
	cfg.EnableHTTPSMonitoring = true
	tr, err := tracer.NewTracer(cfg)
	require.NoError(t, err)
	err = tr.RegisterClient("1")
	require.NoError(t, err)
	defer tr.Stop()

	requestFn := simpleGetRequestsGenerator(t, "127.0.0.1:8001")
	var requests []*nethttp.Request
	for i := 0; i < 100; i++ {
		requests = append(requests, requestFn())
	}

	assertAllRequestsExists2(t, tr, requests)
}

// TestHTTPMonitorLoadWithIncompleteBuffers sends thousands of requests without getting responses for them, in parallel
// we send another request. We expect to capture the another request but not the incomplete requests.
func TestHTTPMonitorLoadWithIncompleteBuffers(t *testing.T) {
	skipTestIfKernelNotSupported(t)
	slowServerAddr := "localhost:8080"
	fastServerAddr := "localhost:8081"

	slowSrvDoneFn := testutil.HTTPServer(t, slowServerAddr, testutil.Options{
		SlowResponse: time.Millisecond * 500, // Half a second.
		WriteTimeout: time.Millisecond * 200,
		ReadTimeout:  time.Millisecond * 200,
	})

	fastSrvDoneFn := testutil.HTTPServer(t, fastServerAddr, testutil.Options{})

	monitor, err := NewMonitor(config.New(), nil, nil)
	require.NoError(t, err)
	require.NoError(t, monitor.Start())
	defer monitor.Stop()

	abortedRequestFn := requestGenerator(t, fmt.Sprintf("%s/ignore", slowServerAddr), emptyBody)
	wg := sync.WaitGroup{}
	abortedRequests := make(chan *nethttp.Request, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := abortedRequestFn()
			abortedRequests <- req
		}()
	}
	fastReq := requestGenerator(t, fastServerAddr, emptyBody)()
	wg.Wait()
	close(abortedRequests)
	slowSrvDoneFn()
	fastSrvDoneFn()

	foundFastReq := false
	// We are iterating for a couple of iterations and making sure the aborted requests will never be found.
	// Since the every call for monitor.GetHTTPStats will delete the pop all entries, and we want to find fastReq
	// then we are using a variable to check if "we ever found it" among the iterations.
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		stats := monitor.GetHTTPStats()
		for req := range abortedRequests {
			requestNotIncluded(t, stats, req)
		}

		foundFastReq = foundFastReq || isRequestIncluded(stats, fastReq)
	}

	require.True(t, foundFastReq)
}

func TestHTTPMonitorIntegrationWithResponseBody(t *testing.T) {
	skipTestIfKernelNotSupported(t)
	targetAddr := "localhost:8080"
	serverAddr := "localhost:8080"

	tests := []struct {
		name            string
		requestBodySize int
	}{
		{
			name:            "no body",
			requestBodySize: 0,
		},
		{
			name:            "1kb body",
			requestBodySize: 1 * kb,
		},
		{
			name:            "10kb body",
			requestBodySize: 10 * kb,
		},
		{
			name:            "100kb body",
			requestBodySize: 100 * kb,
		},
		{
			name:            "500kb body",
			requestBodySize: 500 * kb,
		},
		{
			name:            "2mb body",
			requestBodySize: 2 * mb,
		},
		{
			name:            "10mb body",
			requestBodySize: 10 * mb,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srvDoneFn := testutil.HTTPServer(t, serverAddr, testutil.Options{
				EnableKeepAlives: true,
			})

			monitor, err := NewMonitor(config.New(), nil, nil)
			require.NoError(t, err)
			require.NoError(t, monitor.Start())
			defer monitor.Stop()

			requestFn := requestGenerator(t, targetAddr, bytes.Repeat([]byte("a"), tt.requestBodySize))
			var requests []*nethttp.Request
			for i := 0; i < 100; i++ {
				requests = append(requests, requestFn())
			}
			srvDoneFn()

			assertAllRequestsExists(t, monitor, requests)
		})
	}
}

func TestHTTPMonitorIntegrationSlowResponse(t *testing.T) {
	skipTestIfKernelNotSupported(t)
	targetAddr := "localhost:8080"
	serverAddr := "localhost:8080"

	tests := []struct {
		name                         string
		mapCleanerIntervalSeconds    int
		httpIdleConnectionTTLSeconds int
		slowResponseTime             int
		shouldCapture                bool
	}{
		{
			name:                         "response reaching after cleanup",
			mapCleanerIntervalSeconds:    1,
			httpIdleConnectionTTLSeconds: 1,
			slowResponseTime:             3,
			shouldCapture:                false,
		},
		{
			name:                         "response reaching before cleanup",
			mapCleanerIntervalSeconds:    1,
			httpIdleConnectionTTLSeconds: 3,
			slowResponseTime:             1,
			shouldCapture:                true,
		},
		{
			name:                         "slow response reaching after ttl but cleaner not running",
			mapCleanerIntervalSeconds:    3,
			httpIdleConnectionTTLSeconds: 1,
			slowResponseTime:             2,
			shouldCapture:                true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DD_SYSTEM_PROBE_CONFIG_HTTP_MAP_CLEANER_INTERVAL_IN_S", strconv.Itoa(tt.mapCleanerIntervalSeconds))
			os.Setenv("DD_SYSTEM_PROBE_CONFIG_HTTP_IDLE_CONNECTION_TTL_IN_S", strconv.Itoa(tt.httpIdleConnectionTTLSeconds))

			slowResponseTimeout := time.Duration(tt.slowResponseTime) * time.Second
			serverTimeout := slowResponseTimeout + time.Second
			srvDoneFn := testutil.HTTPServer(t, serverAddr, testutil.Options{
				WriteTimeout: serverTimeout,
				ReadTimeout:  serverTimeout,
				SlowResponse: slowResponseTimeout,
			})

			monitor, err := NewMonitor(config.New(), nil, nil)
			require.NoError(t, err)
			require.NoError(t, monitor.Start())
			defer monitor.Stop()

			// Perform a number of random requests
			req := requestGenerator(t, targetAddr, emptyBody)()
			srvDoneFn()

			// Ensure all captured transactions get sent to user-space
			time.Sleep(10 * time.Millisecond)
			stats := monitor.GetHTTPStats()

			if tt.shouldCapture {
				includesRequest(t, stats, req)
			} else {
				requestNotIncluded(t, stats, req)
			}
		})
	}
}

func TestHTTPMonitorIntegration(t *testing.T) {
	skipTestIfKernelNotSupported(t)

	targetAddr := "localhost:8080"
	serverAddr := "localhost:8080"

	t.Run("with keep-alives", func(t *testing.T) {
		testHTTPMonitor(t, targetAddr, serverAddr, 100, testutil.Options{
			EnableKeepAlives: true,
		})
	})
	t.Run("without keep-alives", func(t *testing.T) {
		testHTTPMonitor(t, targetAddr, serverAddr, 100, testutil.Options{
			EnableKeepAlives: false,
		})
	})
}

func TestHTTPMonitorIntegrationWithNAT(t *testing.T) {
	skipTestIfKernelNotSupported(t)

	// SetupDNAT sets up a NAT translation from 2.2.2.2 to 1.1.1.1
	netlink.SetupDNAT(t)

	targetAddr := "2.2.2.2:8080"
	serverAddr := "1.1.1.1:8080"
	t.Run("with keep-alives", func(t *testing.T) {
		testHTTPMonitor(t, targetAddr, serverAddr, 100, testutil.Options{
			EnableKeepAlives: true,
		})
	})
	t.Run("without keep-alives", func(t *testing.T) {
		testHTTPMonitor(t, targetAddr, serverAddr, 100, testutil.Options{
			EnableKeepAlives: false,
		})
	})
}

func TestUnknownMethodRegression(t *testing.T) {
	skipTestIfKernelNotSupported(t)

	// SetupDNAT sets up a NAT translation from 2.2.2.2 to 1.1.1.1
	netlink.SetupDNAT(t)

	targetAddr := "2.2.2.2:8080"
	serverAddr := "1.1.1.1:8080"
	srvDoneFn := testutil.HTTPServer(t, serverAddr, testutil.Options{
		EnableTLS:        false,
		EnableKeepAlives: true,
	})
	defer srvDoneFn()

	monitor, err := NewMonitor(config.New(), nil, nil)
	require.NoError(t, err)
	err = monitor.Start()
	require.NoError(t, err)
	defer monitor.Stop()

	requestFn := requestGenerator(t, targetAddr, emptyBody)
	for i := 0; i < 100; i++ {
		requestFn()
	}

	time.Sleep(10 * time.Millisecond)
	stats := monitor.GetHTTPStats()

	for key := range stats {
		if key.Method == MethodUnknown {
			t.Error("detected HTTP request with method unknown")
		}
	}

	telemetry := monitor.GetStats()
	require.NotEmpty(t, telemetry)
	_, ok := telemetry["dropped"]
	require.True(t, ok)
	_, ok = telemetry["misses"]
	require.True(t, ok)
}

func TestRSTPacketRegression(t *testing.T) {
	skipTestIfKernelNotSupported(t)

	monitor, err := NewMonitor(config.New(), nil, nil)
	require.NoError(t, err)
	err = monitor.Start()
	require.NoError(t, err)
	defer monitor.Stop()

	serverAddr := "127.0.0.1:8080"
	srvDoneFn := testutil.HTTPServer(t, serverAddr, testutil.Options{
		EnableKeepAlives: true,
	})
	defer srvDoneFn()

	// Create a "raw" TCP socket that will serve as our HTTP client
	// We do this in order to configure the socket option SO_LINGER
	// so we can force a RST packet to be sent during termination
	c, err := net.DialTimeout("tcp", serverAddr, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	// Issue HTTP request
	c.Write([]byte("GET /200/foobar HTTP/1.1\nHost: 127.0.0.1:8080\n\n"))
	io.Copy(io.Discard, c)

	// Configure SO_LINGER to 0 so that triggers an RST when the socket is terminated
	require.NoError(t, c.(*net.TCPConn).SetLinger(0))
	c.Close()
	time.Sleep(100 * time.Millisecond)

	// Assert that the HTTP request was correctly handled despite its forceful termination
	stats := monitor.GetHTTPStats()
	url, err := url.Parse("http://127.0.0.1:8080/200/foobar")
	require.NoError(t, err)
	includesRequest(t, stats, &nethttp.Request{URL: url})
}

func assertAllRequestsExists2(t *testing.T, tracker *tracer.Tracer, requests []*nethttp.Request) {
	requestsExist := make([]bool, len(requests))
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		conns, err := tracker.GetActiveConnections("1")
		require.NoError(t, err)

		for reqIndex, req := range requests {
			for _, httpStats := range conns.HTTP {
				fmt.Println(httpStats, reqIndex, req)
			}
		}
	}

	for reqIndex, exists := range requestsExist {
		require.Truef(t, exists, "request %d was not found (req %v)", reqIndex, requests[reqIndex])
	}
}

func assertAllRequestsExists(t *testing.T, monitor *Monitor, requests []*nethttp.Request) {
	requestsExist := make([]bool, len(requests))
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		stats := monitor.GetHTTPStats()
		for reqIndex, req := range requests {
			requestsExist[reqIndex] = requestsExist[reqIndex] || isRequestIncluded(stats, req)
		}
	}

	for reqIndex, exists := range requestsExist {
		require.Truef(t, exists, "request %d was not found (req %v)", reqIndex, requests[reqIndex])
	}
}

func testHTTPMonitor(t *testing.T, targetAddr, serverAddr string, numReqs int, o testutil.Options) {
	srvDoneFn := testutil.HTTPServer(t, serverAddr, o)

	monitor, err := NewMonitor(config.New(), nil, nil)
	require.NoError(t, err)
	err = monitor.Start()
	require.NoError(t, err)
	defer monitor.Stop()

	// Perform a number of random requests
	requestFn := requestGenerator(t, targetAddr, emptyBody)
	var requests []*nethttp.Request
	for i := 0; i < numReqs; i++ {
		requests = append(requests, requestFn())
	}
	srvDoneFn()

	// Ensure all captured transactions get sent to user-space
	assertAllRequestsExists(t, monitor, requests)
}

var (
	httpMethods         = []string{nethttp.MethodGet, nethttp.MethodHead, nethttp.MethodPost, nethttp.MethodPut, nethttp.MethodPatch, nethttp.MethodDelete, nethttp.MethodOptions}
	httpMethodsWithBody = []string{nethttp.MethodPost, nethttp.MethodPut, nethttp.MethodPatch, nethttp.MethodDelete}
	statusCodes         = []int{nethttp.StatusOK, nethttp.StatusMultipleChoices, nethttp.StatusBadRequest, nethttp.StatusInternalServerError}
)

func simpleGetRequestsGenerator(t *testing.T, targetAddr string) func() *nethttp.Request {
	var (
		random = rand.New(rand.NewSource(time.Now().Unix()))
		idx    = 0
		client = new(nethttp.Client)
	)

	return func() *nethttp.Request {
		idx++
		status := statusCodes[random.Intn(len(statusCodes))]
		req, err := nethttp.NewRequest(nethttp.MethodGet, fmt.Sprintf("https://%s/%d/request-%d", targetAddr, status, idx), nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		require.Equal(t, status, resp.StatusCode)
		return req
	}
}

func requestGenerator(t *testing.T, targetAddr string, reqBody []byte) func() *nethttp.Request {
	var (
		random = rand.New(rand.NewSource(time.Now().Unix()))
		idx    = 0
		client = new(nethttp.Client)
	)

	return func() *nethttp.Request {
		idx++
		var method string
		var body io.Reader
		var finalBody []byte
		if len(reqBody) > 0 {
			finalBody = append([]byte(strings.Repeat(" ", idx)), reqBody...)
			body = bytes.NewReader(finalBody)
			method = httpMethodsWithBody[random.Intn(len(httpMethodsWithBody))]
		} else {
			method = httpMethods[random.Intn(len(httpMethods))]
		}
		status := statusCodes[random.Intn(len(statusCodes))]
		url := fmt.Sprintf("http://%s/%d/request-%d", targetAddr, status, idx)
		req, err := nethttp.NewRequest(method, url, body)
		require.NoError(t, err)

		resp, err := client.Do(req)
		if strings.Contains(targetAddr, "ignore") {
			return req
		}
		require.NoError(t, err)
		if len(reqBody) > 0 {
			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, finalBody, respBody)
		}
		return req
	}
}

func includesRequest(t *testing.T, allStats map[Key]*RequestStats, req *nethttp.Request) {
	if !isRequestIncluded(allStats, req) {
		expectedStatus := testutil.StatusFromPath(req.URL.Path)
		t.Errorf(
			"could not find HTTP transaction matching the following criteria:\n path=%s method=%s status=%d",
			req.URL.Path,
			req.Method,
			expectedStatus,
		)
	}
}

func requestNotIncluded(t *testing.T, allStats map[Key]*RequestStats, req *nethttp.Request) {
	if isRequestIncluded(allStats, req) {
		expectedStatus := testutil.StatusFromPath(req.URL.Path)
		t.Errorf(
			"should not find HTTP transaction matching the following criteria:\n path=%s method=%s status=%d",
			req.URL.Path,
			req.Method,
			expectedStatus,
		)
	}
}

func isRequestIncluded(allStats map[Key]*RequestStats, req *nethttp.Request) bool {
	expectedStatus := testutil.StatusFromPath(req.URL.Path)
	for key, stats := range allStats {
		if key.Path.Content == req.URL.Path && stats.HasStats(expectedStatus) {
			return true
		}
	}

	return false
}
