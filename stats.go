package boomer

import (
	"boomer/internal/builtin"
	"boomer/internal/json"
	"sync/atomic"
	"time"
)

type transaction struct {
	name        string
	success     bool
	elapsedTime int64
	contentSize int64
}

type requestSuccess struct {
	requestType    string
	name           string
	responseTime   int64
	responseLength int64
}

type requestFailure struct {
	requestType  string
	name         string
	responseTime int64
	errMsg       string
}

type requestStats struct {
	entries   map[string]*statsEntry
	errors    map[string]*statsError
	total     *statsEntry
	startTime int64

	transactionChan   chan *transaction
	transactionPassed int64 // accumulated number of passed transactions
	transactionFailed int64 // accumulated number of failed transactions

	requestSuccessChan chan *requestSuccess
	requestFailureChan chan *requestFailure
}

type workerReport struct {
	Errors     map[string]*statsError `json:"errors"`
	State      int32                  `json:"state"`
	Stats      []*statsEntry          `json:"stats"`
	StatsTotal *statsEntry            `json:"stats_total"`
	//Transactions struct {
	//	Failed int `json:"failed"`
	//	Passed int `json:"passed"`
	//} `json:"transactions"`
	UserCount int64 `json:"user_count"`
}

func (r *workerReport) serialize() map[string]interface{} {
	result := make(map[string]interface{})
	marshal, err := json.Marshal(r)
	if err != nil {
		return nil
	}
	err = json.Unmarshal(marshal, &result)
	if err != nil {
		return nil
	}
	return result
}

func newRequestStats() (stats *requestStats) {
	entries := make(map[string]*statsEntry)
	errors := make(map[string]*statsError)

	stats = &requestStats{
		entries: entries,
		errors:  errors,
	}
	stats.transactionChan = make(chan *transaction, 100)
	stats.requestSuccessChan = make(chan *requestSuccess, 100)
	stats.requestFailureChan = make(chan *requestFailure, 100)

	stats.total = &statsEntry{
		Name:   "Total",
		Method: "",
	}
	stats.total.reset()

	return stats
}

func (s *requestStats) logTransaction(name string, success bool, responseTime int64, contentLength int64) {
	if success {
		s.transactionPassed++
	} else {
		s.transactionFailed++
		s.get(name, "transaction").logFailures()
	}
	s.get(name, "transaction").log(responseTime, contentLength)
}

func (s *requestStats) logRequest(method, name string, responseTime int64, contentLength int64) {
	s.total.log(responseTime, contentLength)
	s.get(name, method).log(responseTime, contentLength)
}

func (s *requestStats) logError(method, name, err string) {
	s.total.logFailures()
	s.get(name, method).logFailures()

	// store error in errors map
	key := genMD5(method, name, err)
	entry, ok := s.errors[key]
	if !ok {
		entry = &statsError{
			Name:   name,
			Method: method,
			ErrMsg: err,
		}
		s.errors[key] = entry
	}
	entry.occured()
}

func (s *requestStats) get(name string, method string) (entry *statsEntry) {
	entry, ok := s.entries[name+method]
	if !ok {
		newEntry := &statsEntry{
			Name:          name,
			Method:        method,
			ResponseTimes: make(map[int64]int64),
			NumReqsPerSec: make(map[int64]int64),
			NumFailPerSec: make(map[int64]int64),
		}
		s.entries[name+method] = newEntry
		return newEntry
	}
	return entry
}

func (s *requestStats) clearAll() {
	s.total = &statsEntry{
		Name:   "Total",
		Method: "",
	}
	s.total.reset()
	s.transactionPassed = 0
	s.transactionFailed = 0
	s.entries = make(map[string]*statsEntry)
	s.errors = make(map[string]*statsError)
	s.startTime = time.Now().Unix()
}

func (s *requestStats) serializeStats() []*statsEntry {
	entries := make([]*statsEntry, 0, len(s.entries))
	for _, v := range s.entries {
		if !(v.NumRequests == 0 && v.NumFailures == 0) {
			entries = append(entries, v.clone())
		}
	}
	return entries
}

func (s *requestStats) collectReportData() *workerReport {
	data := &workerReport{
		Errors:     s.errors,
		State:      0,
		Stats:      s.serializeStats(),
		StatsTotal: s.total.clone(),
		UserCount:  0,
	}
	s.errors = make(map[string]*statsError)
	return data
}

// statsEntry represents a single stats entry (Name and Method)
type statsEntry struct {
	// Name (URL) of this stats entry
	Name string `json:"Name"`
	// Method (GET, POST, PUT, etc.)
	Method string `json:"Method"`
	// The number of requests made
	NumRequests int64 `json:"num_requests"`
	// Number of failed request
	NumFailures int64 `json:"num_failures"`
	// Total sum of the response times
	TotalResponseTime int64 `json:"total_response_time"`
	// Minimum response time
	MinResponseTime int64 `json:"min_response_time"`
	// Maximum response time
	MaxResponseTime int64 `json:"max_response_time"`
	// A {response_time => count} dict that holds the response time distribution of all the requests
	// The keys (the response time in ms) are rounded to store 1, 2, ... 9, 10, 20. .. 90,
	// 100, 200 .. 900, 1000, 2000 ... 9000, in order to save memory.
	// This dict is used to calculate the median and percentile response times.
	ResponseTimes map[int64]int64 `json:"response_times"`
	// A {second => request_count} dict that holds the number of requests made per second
	NumReqsPerSec map[int64]int64 `json:"num_reqs_per_sec"`
	// A (second => failure_count) dict that hold the number of failures per second
	NumFailPerSec map[int64]int64 `json:"num_fail_per_sec"`
	// The sum of the content length of all the requests for this entry
	TotalContentLength int64 `json:"total_content_length"`
	// Time of the first request for this entry
	StartTime int64 `json:"start_time"`
	// Time of the last request for this entry
	LastRequestTimestamp int64 `json:"last_request_timestamp"`
	// Boomer doesn't allow None response time for requests like locust.
	// num_none_requests is added to keep compatible with locust.
	NumNoneRequests int64 `json:"num_none_requests"`
}

func (s *statsEntry) resetStartTime() {
	atomic.StoreInt64(&s.StartTime, time.Duration(time.Now().UnixNano()).Milliseconds())
}

func (s *statsEntry) reset() {
	atomic.StoreInt64(&s.StartTime, time.Duration(time.Now().UnixNano()).Milliseconds())
	s.NumRequests = 0
	s.NumFailures = 0
	s.TotalResponseTime = 0
	s.ResponseTimes = make(map[int64]int64)
	s.NumReqsPerSec = make(map[int64]int64)
	s.NumFailPerSec = make(map[int64]int64)
	s.MinResponseTime = 0
	s.MaxResponseTime = 0
	s.LastRequestTimestamp = time.Duration(time.Now().UnixNano()).Milliseconds()
	s.TotalContentLength = 0
}

func (s *statsEntry) log(responseTime int64, contentLength int64) {
	s.NumRequests++

	s.logTimeOfRequest()
	s.logResponseTime(responseTime)

	s.TotalContentLength += contentLength
}

func (s *statsEntry) logTimeOfRequest() {
	s.LastRequestTimestamp = time.Duration(time.Now().UnixNano()).Milliseconds()
	key := time.Now().Unix()
	_, ok := s.NumReqsPerSec[key]
	if !ok {
		s.NumReqsPerSec[key] = 1
	} else {
		s.NumReqsPerSec[key]++
	}
}

func (s *statsEntry) logResponseTime(responseTime int64) {
	s.TotalResponseTime += responseTime

	if s.MinResponseTime == 0 {
		s.MinResponseTime = responseTime
	}

	if responseTime < s.MinResponseTime {
		s.MinResponseTime = responseTime
	}

	if responseTime > s.MaxResponseTime {
		s.MaxResponseTime = responseTime
	}

	var roundedResponseTime int64

	// to avoid too much data that has to be transferred to the master node when
	// running in distributed mode, we save the response time rounded in a dict
	// so that 147 becomes 150, 3432 becomes 3400 and 58760 becomes 59000
	// see also locust's stats.py
	if responseTime < 100 {
		roundedResponseTime = responseTime
	} else if responseTime < 1000 {
		roundedResponseTime = int64(round(float64(responseTime), .5, -1))
	} else if responseTime < 10000 {
		roundedResponseTime = int64(round(float64(responseTime), .5, -2))
	} else {
		roundedResponseTime = int64(round(float64(responseTime), .5, -3))
	}

	_, ok := s.ResponseTimes[roundedResponseTime]
	if !ok {
		s.ResponseTimes[roundedResponseTime] = 1
	} else {
		s.ResponseTimes[roundedResponseTime]++
	}
}

func (s *statsEntry) logFailures() {
	s.NumFailures++

	key := time.Now().Unix()
	_, ok := s.NumFailPerSec[key]
	if !ok {
		s.NumFailPerSec[key] = 1
	} else {
		s.NumFailPerSec[key]++
	}
}

func (s *statsEntry) extend(other *statsEntry) {

	// Extend the data from the current StatsEntry with the stats from another
	// StatsEntry instance.

	if s.LastRequestTimestamp != 0 && other.LastRequestTimestamp != 0 {
		atomic.StoreInt64(&s.LastRequestTimestamp, builtin.MaxInt64(s.LastRequestTimestamp, other.LastRequestTimestamp))
	} else if other.LastRequestTimestamp != 0 {
		atomic.StoreInt64(&s.LastRequestTimestamp, other.LastRequestTimestamp)
	}
	if s.MinResponseTime != 0 && other.MinResponseTime != 0 {
		atomic.StoreInt64(&s.MinResponseTime, builtin.MinInt64(s.MinResponseTime, other.MinResponseTime))
	} else if other.MinResponseTime != 0 {
		atomic.StoreInt64(&s.MinResponseTime, other.MinResponseTime)
	}

	atomic.StoreInt64(&s.StartTime, builtin.MinInt64(s.StartTime, other.StartTime))
	atomic.StoreInt64(&s.MaxResponseTime, builtin.MaxInt64(s.MaxResponseTime, other.MaxResponseTime))
	atomic.AddInt64(&s.NumNoneRequests, other.NumNoneRequests)
	atomic.AddInt64(&s.NumRequests, other.NumRequests)
	atomic.AddInt64(&s.NumFailures, other.NumFailures)
	atomic.AddInt64(&s.TotalResponseTime, other.TotalResponseTime)
	atomic.AddInt64(&s.TotalContentLength, other.TotalContentLength)

	for k, v := range other.ResponseTimes {
		s.ResponseTimes[k] += v
	}

	for k, v := range other.NumReqsPerSec {
		s.NumReqsPerSec[k] += v
	}

	for k, v := range other.NumFailPerSec {
		s.NumFailPerSec[k] += v
	}
}

func (s *statsEntry) clone() *statsEntry {
	newStats := &statsEntry{
		Name:                 s.Name,
		Method:               s.Method,
		NumRequests:          s.NumRequests,
		NumFailures:          s.NumFailures,
		TotalResponseTime:    s.TotalResponseTime,
		MinResponseTime:      s.MinResponseTime,
		MaxResponseTime:      s.MaxResponseTime,
		ResponseTimes:        s.ResponseTimes,
		NumReqsPerSec:        s.NumReqsPerSec,
		NumFailPerSec:        s.NumFailPerSec,
		TotalContentLength:   s.TotalContentLength,
		StartTime:            s.StartTime,
		LastRequestTimestamp: s.LastRequestTimestamp,
		NumNoneRequests:      s.NumNoneRequests,
	}
	s.reset()
	return newStats
}

type statsError struct {
	Name        string `json:"name"`
	Method      string `json:"method"`
	ErrMsg      string `json:"errMsg"`
	Occurrences int64  `json:"occurrences"`
}

func (err *statsError) occured() {
	err.Occurrences++
}

func (err *statsError) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	m["method"] = err.Method
	m["name"] = err.Name
	m["error"] = err.ErrMsg
	m["occurrences"] = err.Occurrences
	return m
}
