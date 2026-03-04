package autodiscovery

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"vigil/internal/evaluator"
	"vigil/internal/models"

	"gorm.io/gorm"
)

type Discovery struct {
	db         *gorm.DB
	lokiClient *evaluator.LokiClient
	interval   time.Duration
}

func New(db *gorm.DB, lokiClient *evaluator.LokiClient, interval time.Duration) *Discovery {
	return &Discovery{
		db:         db,
		lokiClient: lokiClient,
		interval:   interval,
	}
}

// Run starts the auto-discovery loop. Blocks until ctx is cancelled.
func (d *Discovery) Run(ctx context.Context) {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	// Run first scan after a short delay
	time.Sleep(5 * time.Second)
	d.scanAll()

	for {
		select {
		case <-ctx.Done():
			log.Println("autodiscovery stopping")
			return
		case <-ticker.C:
			d.scanAll()
		}
	}
}

func (d *Discovery) scanAll() {
	var rules []models.AutoDiscoveryRule
	if err := d.db.Where("active = ?", true).Find(&rules).Error; err != nil {
		log.Printf("autodiscovery: failed to load rules: %v", err)
		return
	}

	for i := range rules {
		d.scanRule(&rules[i])
	}
}

func (d *Discovery) scanRule(rule *models.AutoDiscoveryRule) {
	patterns, err := d.lokiClient.GetPatterns(rule.LokiSelector)
	if err != nil {
		log.Printf("autodiscovery: pattern scan failed for %q: %v", rule.LokiSelector, err)
		return
	}

	var patternFilter *regexp.Regexp
	if rule.Pattern != "" {
		// Convert simple glob to regex
		regexPattern := strings.ReplaceAll(rule.Pattern, "*", ".*")
		patternFilter, err = regexp.Compile(regexPattern)
		if err != nil {
			log.Printf("autodiscovery: invalid pattern %q: %v", rule.Pattern, err)
			return
		}
	}

	for _, p := range patterns {
		if p.Count < rule.MinSamples {
			continue
		}

		if patternFilter != nil && !patternFilter.MatchString(p.Pattern) {
			continue
		}

		d.ensureSwitch(rule, p)
	}

	// Update last scan timestamp
	d.db.Model(rule).Update("last_scan_at", time.Now())
}

func (d *Discovery) ensureSwitch(rule *models.AutoDiscoveryRule, pattern evaluator.PatternResult) {
	// Generate a unique name from the selector and pattern
	name := fmt.Sprintf("auto_%s", sanitizeName(pattern.Pattern))

	// Check if switch already exists
	var count int64
	d.db.Model(&models.Switch{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return
	}

	// Build a LogQL query that matches this pattern
	query := fmt.Sprintf(`%s |= %q`, rule.LokiSelector, pattern.Pattern)

	sw := models.Switch{
		Name:                name,
		Signal:              models.SignalLoki,
		Query:               query,
		Mode:                models.ModeIrregularity,
		State:               models.StateLearning,
		AutoCreated:         true,
		MinSamples:          rule.MinSamples,
		ToleranceMultiplier: rule.ToleranceMultiplier,
		StateChangedAt:      time.Now(),
	}

	if err := d.db.Create(&sw).Error; err != nil {
		log.Printf("autodiscovery: failed to create switch %q: %v", name, err)
		return
	}

	log.Printf("autodiscovery: created switch %q from pattern %q", name, pattern.Pattern)
}

// sanitizeName converts a pattern into a safe switch name
func sanitizeName(pattern string) string {
	// Replace non-alphanumeric chars with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	name := re.ReplaceAllString(pattern, "_")
	name = strings.Trim(name, "_")

	// Truncate to reasonable length
	if len(name) > 80 {
		name = name[:80]
	}

	return strings.ToLower(name)
}
