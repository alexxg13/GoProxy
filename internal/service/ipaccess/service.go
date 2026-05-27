package ipaccess

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"sync"
	"time"

	"GoProxy/internal/models"
	iputils "GoProxy/pkg/ip-utils"
	"GoProxy/pkg/lrucache"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Decision struct {
	Allowed         bool
	Reason          models.DenyReason
	Grey            bool
	CaptchaVerified bool
}

type cachedDecision struct {
	decision Decision
}

type fileConfig struct {
	DefaultPolicy string   `yaml:"default_policy"`
	Allow         []string `yaml:"allow"`
	Deny          []string `yaml:"deny"`
	Grey          []string `yaml:"grey"`
}
type IPService interface {
	Check(ctx context.Context, ipStr string) Decision
	VerifyCaptcha(ctx context.Context, ipStr string, answer string) Decision
	List(ruleType models.RuleType) []models.IPRule
	Create(ctx context.Context, req models.IPRuleCreate) (models.IPRule, error)
	Delete(ctx context.Context, id string) error
}
type IPServiceImpl struct {
	mu            sync.RWMutex
	rules         map[string]models.IPRule
	denySet       *iputils.RuleSet[string]
	allowSet      *iputils.RuleSet[string]
	greySet       *iputils.RuleSet[string]
	defaultDeny   bool
	cache         *lrucache.Cache[string, cachedDecision]
	cacheCapacity int
	cacheTTL      time.Duration
	captchaTTL    time.Duration
	captchaPass   map[string]time.Time
	configPath    string
	requestCounts map[string]int64
}

type Config struct {
	ConfigPath    string
	CacheCapacity int
	CacheTTL      time.Duration
	CaptchaTTL    time.Duration
	DefaultDeny   bool
}

func New(cfg Config) (*IPServiceImpl, error) {
	capacity := cfg.CacheCapacity
	if capacity <= 0 {
		capacity = 4096
	}
	captchaTTL := cfg.CaptchaTTL
	if captchaTTL <= 0 {
		captchaTTL = 10 * time.Minute
	}
	s := &IPServiceImpl{
		rules:         make(map[string]models.IPRule),
		denySet:       iputils.NewRuleSet[string](),
		allowSet:      iputils.NewRuleSet[string](),
		greySet:       iputils.NewRuleSet[string](),
		defaultDeny:   cfg.DefaultDeny,
		cache:         lrucache.New[string, cachedDecision](capacity, cfg.CacheTTL),
		cacheCapacity: capacity,
		cacheTTL:      cfg.CacheTTL,
		captchaTTL:    captchaTTL,
		captchaPass:   make(map[string]time.Time),
		configPath:    cfg.ConfigPath,
		requestCounts: make(map[string]int64),
	}

	if cfg.ConfigPath != "" {
		if err := s.loadFromFile(cfg.ConfigPath); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		go s.watchConfig(cfg.ConfigPath)
	}

	return s, nil
}

func (s *IPServiceImpl) Check(ctx context.Context, ipStr string) Decision {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return Decision{Allowed: false, Reason: models.DenyNotWhitelisted}
	}
	addr = addr.Unmap()

	if cached, ok := s.cache.Get(ipStr); ok {
		return cached.decision
	}

	decision := s.evaluate(addr)
	if !decision.Grey && !decision.CaptchaVerified {
		s.cache.Set(ipStr, cachedDecision{decision: decision})
	}
	return decision
}

func (s *IPServiceImpl) VerifyCaptcha(ctx context.Context, ipStr string, answer string) Decision {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return Decision{Allowed: false, Reason: models.DenyNotWhitelisted}
	}
	addr = addr.Unmap()

	s.mu.Lock()
	defer s.mu.Unlock()

	if ruleID, ok := s.denySet.Match(addr); ok {
		s.requestCounts[ruleID]++
		return Decision{Allowed: false, Reason: models.DenyBlacklisted}
	}
	if ruleID, ok := s.allowSet.Match(addr); ok {
		s.requestCounts[ruleID]++
		return Decision{Allowed: true}
	}
	if _, ok := s.greySet.Match(addr); !ok {
		if s.defaultDeny {
			return Decision{Allowed: false, Reason: models.DenyNotWhitelisted}
		}
		return Decision{Allowed: true}
	}
	if answer != "4" {
		return Decision{Allowed: false, Grey: true, Reason: models.DenyGreyCaptcha}
	}

	s.captchaPass[addr.String()] = time.Now().Add(s.captchaTTL)
	s.resetCache()
	return Decision{Allowed: true, CaptchaVerified: true}
}

func (s *IPServiceImpl) evaluate(addr netip.Addr) Decision {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ruleID, ok := s.denySet.Match(addr); ok {
		s.requestCounts[ruleID]++
		return Decision{Allowed: false, Reason: models.DenyBlacklisted}
	}

	if ruleID, ok := s.allowSet.Match(addr); ok {
		s.requestCounts[ruleID]++
		return Decision{Allowed: true}
	}

	if _, ok := s.greySet.Match(addr); ok {
		key := addr.String()
		if expiresAt, verified := s.captchaPass[key]; verified {
			if time.Now().Before(expiresAt) {
				return Decision{Allowed: true, CaptchaVerified: true}
			}
			delete(s.captchaPass, key)
		}
		return Decision{Allowed: false, Grey: true, Reason: models.DenyGreyCaptcha}
	}

	if s.defaultDeny {
		return Decision{Allowed: false, Reason: models.DenyNotWhitelisted}
	}
	return Decision{Allowed: true}
}

func (s *IPServiceImpl) List(ruleType models.RuleType) []models.IPRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]models.IPRule, 0, len(s.rules))
	for _, r := range s.rules {
		if ruleType != "" && r.Type != ruleType {
			continue
		}
		r.Requests = s.requestCounts[r.ID]
		out = append(out, r)
	}
	return out
}

func (s *IPServiceImpl) Create(ctx context.Context, req models.IPRuleCreate) (models.IPRule, error) {
	if _, err := iputils.Parse(req.IP); err != nil {
		return models.IPRule{}, fmt.Errorf("invalid IP: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("ipr%d", time.Now().UnixNano())
	rule := models.IPRule{
		ID:        id,
		IP:        req.IP,
		Type:      req.Type,
		Reason:    req.Reason,
		AddedDate: time.Now().UTC(),
	}
	s.rules[id] = rule
	s.insertRule(rule)
	s.resetCache()
	s.persistLocked()

	return rule, nil
}

func (s *IPServiceImpl) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, ok := s.rules[id]
	if !ok {
		return fmt.Errorf("rule not found")
	}

	delete(s.rules, id)
	s.rebuildSetsLocked()
	s.resetCache()
	s.persistLocked()
	_ = rule
	return nil
}

func (s *IPServiceImpl) insertRule(rule models.IPRule) {
	prefixes, err := iputils.Prefixes(rule.IP)
	if err != nil {
		return
	}
	prefixes = iputils.ExpandLoopbackPrefixes(prefixes)
	switch rule.Type {
	case models.RuleDeny:
		s.denySet.InsertPrefixes(prefixes, 0, rule.ID)
	case models.RuleAllow:
		s.allowSet.InsertPrefixes(prefixes, 0, rule.ID)
	case models.RuleGrey:
		s.greySet.InsertPrefixes(prefixes, 0, rule.ID)
	}
}

func (s *IPServiceImpl) rebuildSetsLocked() {
	s.denySet = iputils.NewRuleSet[string]()
	s.allowSet = iputils.NewRuleSet[string]()
	s.greySet = iputils.NewRuleSet[string]()
	for _, rule := range s.rules {
		s.insertRule(rule)
	}
}

func (s *IPServiceImpl) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.rules = make(map[string]models.IPRule)
	s.defaultDeny = cfg.DefaultPolicy != "allow"

	for i, ip := range cfg.Deny {
		id := fmt.Sprintf("cfg-deny-%d", i)
		rule := models.IPRule{ID: id, IP: ip, Type: models.RuleDeny, Reason: "config", AddedDate: time.Now().UTC()}
		s.rules[id] = rule
	}
	for i, ip := range cfg.Allow {
		id := fmt.Sprintf("cfg-allow-%d", i)
		rule := models.IPRule{ID: id, IP: ip, Type: models.RuleAllow, Reason: "config", AddedDate: time.Now().UTC()}
		s.rules[id] = rule
	}
	for i, ip := range cfg.Grey {
		id := fmt.Sprintf("cfg-grey-%d", i)
		rule := models.IPRule{ID: id, IP: ip, Type: models.RuleGrey, Reason: "config", AddedDate: time.Now().UTC()}
		s.rules[id] = rule
	}

	s.rebuildSetsLocked()
	return nil
}

func (s *IPServiceImpl) persistLocked() {
	if s.configPath == "" {
		return
	}

	cfg := fileConfig{DefaultPolicy: "deny"}
	if !s.defaultDeny {
		cfg.DefaultPolicy = "allow"
	}

	for _, r := range s.rules {
		switch r.Type {
		case models.RuleAllow:
			cfg.Allow = append(cfg.Allow, r.IP)
		case models.RuleDeny:
			cfg.Deny = append(cfg.Deny, r.IP)
		case models.RuleGrey:
			cfg.Grey = append(cfg.Grey, r.IP)
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return
	}
	_ = os.WriteFile(s.configPath, data, 0o644)
}

func (s *IPServiceImpl) resetCache() {
	s.cache = lrucache.New[string, cachedDecision](s.cacheCapacity, s.cacheTTL)
}

func (s *IPServiceImpl) watchConfig(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		return
	}

	var reloadTimer *time.Timer
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			if reloadTimer != nil {
				reloadTimer.Stop()
			}
			reloadTimer = time.AfterFunc(300*time.Millisecond, func() {
				if err := s.loadFromFile(path); err == nil {
					s.mu.Lock()
					s.resetCache()
					s.mu.Unlock()
				}
			})
		case <-watcher.Errors:
		}
	}
}
