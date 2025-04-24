package jira

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

type RateLimitInfo struct {
	Data map[string]int
}

func (m *RateLimitInfo) JiraBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if resp != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			nodeId, isSet := getResponseHeaderString(resp, "x-anodeid")
			if !isSet {
				return simpleBackOff(min, max, attemptNum)
			}
			interval, isSet := getResponseHeaderInt(resp, "x-ratelimit-interval-seconds")
			if !isSet {
				return simpleBackOff(min, max, attemptNum)
			}
			fillRate, isSet := getResponseHeaderInt(resp, "x-ratelimit-fillrate")
			if !isSet {
				return simpleBackOff(min, max, attemptNum)
			}
			retryAfter, isSet := getResponseHeaderInt(resp, "Retry-After")
			if !isSet {
				return simpleBackOff(min, max, attemptNum)
			}
			m.Data[nodeId] = retryAfter
			if sumMapValues(m.Data) == 0 {
				rateLimit := interval / fillRate
				return time.Duration(rateLimit) * time.Second
			}
		}
	}
	return simpleBackOff(min, max, attemptNum)
}

func getResponseHeaderInt(resp *http.Response, key string) (int, bool) {
	if resp.Header.Get(key) != "" {
		value, err := strconv.Atoi(resp.Header.Get(key))
		if err == nil {
			return value, true
		}
	}
	return 0, false
}

func getResponseHeaderString(resp *http.Response, key string) (string, bool) {
	if resp.Header.Get(key) != "" {
		return resp.Header.Get(key), true
	}
	return "", false
}

func sumMapValues(m map[string]int) int {
	var sum int
	for _, value := range m {
		sum += value
	}
	return sum
}

func simpleBackOff(min, max time.Duration, attemptNum int) time.Duration {
	mult := math.Pow(2, float64(attemptNum)) * float64(min)
	sleep := time.Duration(mult)
	if float64(sleep) != mult || sleep > max {
		sleep = max
	}
	return sleep
}
