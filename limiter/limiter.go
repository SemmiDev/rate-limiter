package limiter

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrTooManyRequests  = errors.New("too many requests")
	ErrRateLimiterCheck = errors.New("failed to perform rate limiting check")
	ErrNoMatchingRule   = errors.New("no matching rule found")
	ErrNoRateLimit      = errors.New("no rate limit found")
)

type Header struct {
	XRatelimitRemaining  int `json:"X-Ratelimit-Remaining"`
	XRatelimit           int `json:"X-Ratelimit"`
	XRateLimitRetryAfter int `json:"X-Ratelimit-Retry-After"`
}

type Response struct {
	*Header
	Limited bool
	Error   error `json:"error"`
}

var mutex sync.RWMutex

func IsRateLimited(r *http.Request) *Response {
	domain := r.Header.Get("domain")
	descriptors := make(map[string]string)

	for key, descriptor := range r.Header {
		if len(descriptor) > 0 {
			descriptors[key] = descriptor[0]
		}
	}

	mutex.RLock()
	defer mutex.RUnlock()

	var header Header
	var ruleMatched bool

	for _, rule := range rules {
		if rule.Domain == domain && matchDescriptors(rule.Descriptors, descriptors) {
			ruleMatched = true

			if rule.RateLimit != nil {
				key := getRedisKey(rule, rule.Descriptors, r.Header.Get(rule.BasedOn))
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// increment the key, if not exists, create it
				val, err := RedisClient.Incr(ctx, key).Result()
				if err != nil {
					log.Error().Err(err).Msg("Failed to increment rate limiting key")
					return &Response{
						Header:  nil,
						Limited: false,
						Error:   ErrRateLimiterCheck,
					}
				}

				// if limit is reached, return error
				if val > int64(rule.RateLimit.RequestsPerUnit) {
					ttl, err := RedisClient.TTL(ctx, key).Result()
					if err != nil {
						log.Error().Err(err).Msg("Failed to get rate limiting key TTL")
						return &Response{
							Header:  nil,
							Limited: false,
							Error:   ErrRateLimiterCheck,
						}
					}

					header.XRatelimitRemaining = 0
					header.XRatelimit = rule.RateLimit.RequestsPerUnit
					header.XRateLimitRetryAfter = int(ttl.Seconds())

					return &Response{
						Header:  &header,
						Limited: true,
						Error:   ErrTooManyRequests,
					}
				}

				// if the key is new, set the expiration
				if val == 1 {
					duration := getExpirationDuration(rule.RateLimit.Unit, rule.RateLimit.Multiplier)
					_, err := RedisClient.Expire(context.Background(), key, duration).Result()
					if err != nil {
						log.Error().Err(err).Msg("Failed to set rate limiting key expiration")
					}

					header.XRatelimitRemaining = rule.RateLimit.RequestsPerUnit - 1
					header.XRatelimit = rule.RateLimit.RequestsPerUnit
					header.XRateLimitRetryAfter = 0

					return &Response{
						Header:  &header,
						Limited: false,
						Error:   nil,
					}
				}

				header.XRatelimitRemaining = rule.RateLimit.RequestsPerUnit - int(val)
				header.XRatelimit = rule.RateLimit.RequestsPerUnit
				header.XRateLimitRetryAfter = 0

				return &Response{
					Header:  &header,
					Limited: false,
					Error:   nil,
				}
			}
		}
	}

	if !ruleMatched {
		return &Response{
			Header:  nil,
			Limited: false,
			Error:   ErrNoMatchingRule,
		}
	}

	return &Response{
		Header:  nil,
		Limited: false,
		Error:   ErrNoRateLimit,
	}
}

func matchDescriptors(ruleDescriptors []Descriptor, requestDescriptors map[string]string) bool {
	requestDescriptors = toLowerKeys(requestDescriptors)

	for _, ruleDesc := range ruleDescriptors {
		requestValue, ok := requestDescriptors[ruleDesc.Key]
		if !ok || requestValue != ruleDesc.Value {
			return false
		} else {
			log.Info().Msgf("Matched descriptor: %s", ruleDesc.Key)
		}
	}

	return true
}

func toLowerKeys(requestDescriptors map[string]string) map[string]string {
	lowerCaseMap := make(map[string]string)
	for key, value := range requestDescriptors {
		key = strings.ToLower(key)
		lowerCaseMap[key] = value
	}
	return lowerCaseMap
}

func getRedisKey(rule Rule, descriptors []Descriptor, userID string) string {
	descriptorValues := make([]string, len(descriptors))

	for i, descriptor := range descriptors {
		descriptorValues[i] = descriptor.Value
	}

	descriptorStr := strings.Join(descriptorValues, "#")
	return fmt.Sprintf("%s#%s#%s", rule.Domain, descriptorStr, userID)
}

func getExpirationDuration(unit string, multiplier int) time.Duration {
	switch unit {
	case "second":
		return time.Second * time.Duration(multiplier)
	case "minute":
		return time.Minute * time.Duration(multiplier)
	case "hour":
		return time.Hour * time.Duration(multiplier)
	case "day":
		return time.Hour * 24 * time.Duration(multiplier)
	default:
		return time.Minute
	}
}
