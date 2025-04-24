package jira

import (
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"
)

func TestSumMapValues(t *testing.T) {
	var testCases = []struct {
		name   string
		m      map[string]int
		expect int
	}{
		{
			name:   "NilMap",
			m:      nil,
			expect: 0,
		},
		{
			name:   "EmptyMap",
			m:      map[string]int{},
			expect: 0,
		},
		{
			name: "SingleEntryZero",
			m: map[string]int{
				"node-1": 0,
			},
			expect: 0,
		},
		{
			name: "SingleEntryNonZero",
			m: map[string]int{
				"node-1": 2,
			},
			expect: 2,
		},
		{
			name: "MultipleEntriesZero",
			m: map[string]int{
				"node-1": 0,
				"node-2": 0,
			},
			expect: 0,
		},
		{
			name: "MultipleEntriesNonZero",
			m: map[string]int{
				"node-1": 1,
				"node-2": 2,
			},
			expect: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := sumMapValues(tc.m)
			if got != tc.expect {
				t.Errorf("expected: %d, but got: %d", tc.expect, got)
			}
		})
	}
}

func TestSimpleBackoff(t *testing.T) {
	retryWaitMin := 1 * time.Second
	retryWaitMax := 30 * time.Second
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("Attempt-%d", i), func(t *testing.T) {
			got := simpleBackOff(retryWaitMin, retryWaitMax, i)
			expect := time.Duration(math.Pow(2, float64(i)) * float64(retryWaitMin))
			if expect >= retryWaitMax {
				expect = retryWaitMax
			}
			if got != expect {
				t.Errorf("expect: %s, but got: %s", expect, got)
			}
		})
	}
}

func TestJiraBackoff(t *testing.T) {
	var testCases = []struct {
		name         string
		data         map[string]int
		retryWaitMin time.Duration
		retryWaitMax time.Duration
		attempt      int
		response     *http.Response
		expect       time.Duration
	}{
		{
			name:         "NilData",
			data:         nil,
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response:     &http.Response{StatusCode: 200},
			expect:       1 * time.Second,
		},
		{
			name:         "NilResponse",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response:     nil,
			expect:       1 * time.Second,
		},
		{
			name:         "200Response",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response:     &http.Response{StatusCode: 200},
			expect:       1 * time.Second,
		},
		{
			name:         "429ResponseMissingHeaders",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response:     &http.Response{StatusCode: 429},
			expect:       1 * time.Second,
		},
		{
			name:         "429ResponseMissingNodeIdHeader",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response: &http.Response{
				Header:     http.Header{},
				StatusCode: 429,
			},
			expect: 1 * time.Second,
		},
		{
			name:         "429ResponseMissingIntervalHeader",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response: &http.Response{
				Header: http.Header{
					"X-Anodeid": []string{
						"node-1",
					},
				},
				StatusCode: 429,
			},
			expect: 1 * time.Second,
		},
		{
			name:         "429ResponseMissingFillRateHeader",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response: &http.Response{
				Header: http.Header{
					"X-Anodeid": []string{
						"node-1",
					},
					"X-Ratelimit-Interval-Seconds": []string{
						"60",
					},
				},
				StatusCode: 429,
			},
			expect: 1 * time.Second,
		},
		{
			name:         "429ResponseMissingRetryAfterHeader",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response: &http.Response{
				Header: http.Header{
					"X-Anodeid": []string{
						"node-1",
					},
					"X-Ratelimit-Interval-Seconds": []string{
						"60",
					},
					"X-Ratelimit-Fillrate": []string{
						"15",
					},
				},
				StatusCode: 429,
			},
			expect: 1 * time.Second,
		},
		{
			name:         "429ResponseWithRateLimitHeadersRetryAfter",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response: &http.Response{
				Header: http.Header{
					"X-Anodeid": []string{
						"node-1",
					},
					"X-Ratelimit-Interval-Seconds": []string{
						"60",
					},
					"X-Ratelimit-Fillrate": []string{
						"15",
					},
					"Retry-After": []string{
						"2",
					},
				},
				StatusCode: 429,
			},
			expect: 1 * time.Second,
		},
		{
			name:         "429ResponseWithRateLimitHeadersRetryAfterFirstAttempt",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      1,
			response: &http.Response{
				Header: http.Header{
					"X-Anodeid": []string{
						"node-1",
					},
					"X-Ratelimit-Interval-Seconds": []string{
						"60",
					},
					"X-Ratelimit-Fillrate": []string{
						"15",
					},
					"Retry-After": []string{
						"2",
					},
				},
				StatusCode: 429,
			},
			expect: 2 * time.Second,
		},
		{
			name:         "429ResponseWithRateLimitHeadersRetryAfterSecondAttempt",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      2,
			response: &http.Response{
				Header: http.Header{
					"X-Anodeid": []string{
						"node-1",
					},
					"X-Ratelimit-Interval-Seconds": []string{
						"60",
					},
					"X-Ratelimit-Fillrate": []string{
						"15",
					},
					"Retry-After": []string{
						"2",
					},
				},
				StatusCode: 429,
			},
			expect: 4 * time.Second,
		},
		{
			name:         "429ResponseWithRateLimitHeadersRetryAfterThirdAttempt",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      3,
			response: &http.Response{
				Header: http.Header{
					"X-Anodeid": []string{
						"node-1",
					},
					"X-Ratelimit-Interval-Seconds": []string{
						"60",
					},
					"X-Ratelimit-Fillrate": []string{
						"15",
					},
					"Retry-After": []string{
						"2",
					},
				},
				StatusCode: 429,
			},
			expect: 8 * time.Second,
		},
		{
			name:         "429ResponseWithRateLimitHeaders",
			data:         map[string]int{},
			retryWaitMin: 1 * time.Second,
			retryWaitMax: 30 * time.Second,
			attempt:      0,
			response: &http.Response{
				Header: http.Header{
					"X-Anodeid": []string{
						"node-1",
					},
					"X-Ratelimit-Interval-Seconds": []string{
						"60",
					},
					"X-Ratelimit-Fillrate": []string{
						"15",
					},
					"Retry-After": []string{
						"0",
					},
				},
				StatusCode: 429,
			},
			// This value is equal to X-Ratelimit-Interval-Seconds / X-Ratelimit-Fillrate
			expect: 4 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rateLimitInfo := &RateLimitInfo{Data: tc.data}

			got := rateLimitInfo.JiraBackoff(tc.retryWaitMin, tc.retryWaitMax, tc.attempt, tc.response)
			if got != tc.expect {
				t.Errorf("expected: %d, but got: %d", tc.expect, got)
			}
		})
	}
}
