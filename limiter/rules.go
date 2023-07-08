package limiter

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var rules []Rule

type Rule struct {
	Domain      string       `yaml:"domain"`
	BasedOn     string       `yaml:"based_on"`
	Descriptors []Descriptor `yaml:"descriptors"`
	RateLimit   *RateLimit   `yaml:"rate_limit,omitempty"`
}

type Descriptor struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type RateLimit struct {
	Unit            string `yaml:"unit"`
	Multiplier      int    `yaml:"multiplier"`
	RequestsPerUnit int    `yaml:"requests_per_unit"`
}

func LoadRules(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var updatedRules []Rule
	err = yaml.Unmarshal(file, &updatedRules)
	if err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	err = validateRules(updatedRules)
	if err != nil {
		return fmt.Errorf("failed to validate rules: %w", err)
	}

	rules = updatedRules

	return nil
}

func PrintRules(rules []Rule) {
	for _, v := range rules {
		fmt.Println("total rules: ", len(rules))
		fmt.Println("domain: ", v.Domain)
		fmt.Println("descriptors: ", v.Descriptors)
		fmt.Println("rate limit: ", v.RateLimit)
	}
}

func ReloadRulesPeriodically(filename string, interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			err := LoadRules(filename)
			if err != nil {
				log.Warn().Err(err).Msg("failed to load rules file")
			}
			log.Info().Msg("Reloaded rules")
		}
	}()
}

func validateRules(rules []Rule) error {
	for _, rule := range rules {
		if rule.Domain == "" {
			return fmt.Errorf("domain is required for rate limiting rule")
		}

		if rule.BasedOn == "" {
			return fmt.Errorf("based_on is required for rate limit in domain %s", rule.Domain)
		}

		if len(rule.Descriptors) == 0 {
			return fmt.Errorf("at least one descriptor is required for rate limiting rule in domain %s", rule.Domain)
		}

		for _, descriptor := range rule.Descriptors {
			if descriptor.Key == "" || descriptor.Value == "" {
				return fmt.Errorf("key and value are required for descriptor in domain %s", rule.Domain)
			}
		}

		if rule.RateLimit != nil {
			if rule.RateLimit.Multiplier <= 0 {
				return fmt.Errorf("multiplier must be greater than 0 in rate limit for domain %s", rule.Domain)
			}
			if rule.RateLimit.Unit == "" {
				return fmt.Errorf("unit is required for rate limit in domain %s", rule.Domain)
			}
			if rule.RateLimit.RequestsPerUnit <= 0 {
				return fmt.Errorf("requests_per_unit must be greater than 0 in rate limit for domain %s", rule.Domain)
			}
		}
	}

	return nil
}
