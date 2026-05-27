package ipaccess

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"GoProxy/internal/models"
)

func TestServiceCheckDenyAllowDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	_ = os.WriteFile(path, []byte(`default_policy: deny
allow:
  - 192.168.1.1
deny:
  - 10.0.0.0/8
grey: []
`), 0o644)

	s, err := New(Config{ConfigPath: path, DefaultDeny: true, CacheCapacity: 100, CacheTTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}

	if d := s.Check(context.Background(), "10.1.2.3"); d.Allowed {
		t.Fatal("deny list should block")
	}
	if d := s.Check(context.Background(), "192.168.1.1"); !d.Allowed {
		t.Fatal("allow list should permit")
	}
	if d := s.Check(context.Background(), "8.8.8.8"); d.Allowed {
		t.Fatal("default deny should block unknown IP")
	}
}

func TestServiceCRUD(t *testing.T) {
	s, err := New(Config{DefaultDeny: true, CacheCapacity: 100})
	if err != nil {
		t.Fatal(err)
	}

	rule, err := s.Create(context.Background(), models.IPRuleCreate{
		IP:     "203.0.113.0/24",
		Type:   models.RuleAllow,
		Reason: "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	if d := s.Check(context.Background(), "203.0.113.50"); !d.Allowed {
		t.Fatal("created allow rule should permit")
	}

	rules := s.List(models.RuleAllow)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	if err := s.Delete(context.Background(), rule.ID); err != nil {
		t.Fatal(err)
	}
	if d := s.Check(context.Background(), "203.0.113.50"); d.Allowed {
		t.Fatal("deleted rule should not permit")
	}
}

func TestServiceLoopbackIPv6(t *testing.T) {
	s, err := New(Config{DefaultDeny: true, CacheCapacity: 100})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Create(context.Background(), models.IPRuleCreate{
		IP: "127.0.0.1", Type: models.RuleAllow, Reason: "localhost",
	})
	if err != nil {
		t.Fatal(err)
	}

	if d := s.Check(context.Background(), "::1"); !d.Allowed {
		t.Fatal("::1 should match allow rule for 127.0.0.1")
	}
}

func TestServiceGreylist(t *testing.T) {
	s, err := New(Config{DefaultDeny: false})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Create(context.Background(), models.IPRuleCreate{
		IP: "1.2.3.4", Type: models.RuleGrey, Reason: "captcha",
	})
	if err != nil {
		t.Fatal(err)
	}

	d := s.Check(context.Background(), "1.2.3.4")
	if d.Allowed || !d.Grey {
		t.Fatal("grey list should require captcha")
	}
}

func TestServiceGreylistAllowsAfterCaptcha(t *testing.T) {
	s, err := New(Config{DefaultDeny: false, CaptchaTTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Create(context.Background(), models.IPRuleCreate{
		IP: "1.2.3.4", Type: models.RuleGrey, Reason: "captcha",
	})
	if err != nil {
		t.Fatal(err)
	}

	if d := s.VerifyCaptcha(context.Background(), "1.2.3.4", "3"); d.Allowed {
		t.Fatal("wrong captcha answer should not pass")
	}
	if d := s.VerifyCaptcha(context.Background(), "1.2.3.4", "4"); !d.Allowed {
		t.Fatalf("correct captcha answer should pass, got reason %s", d.Reason)
	}
	if d := s.Check(context.Background(), "1.2.3.4"); !d.Allowed {
		t.Fatalf("verified grey IP should be allowed, got reason %s", d.Reason)
	}
}

func TestServiceCaptchaDoesNotBypassDeny(t *testing.T) {
	s, err := New(Config{DefaultDeny: false, CaptchaTTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Create(context.Background(), models.IPRuleCreate{
		IP: "1.2.3.4", Type: models.RuleDeny, Reason: "blocked",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Create(context.Background(), models.IPRuleCreate{
		IP: "1.2.3.4", Type: models.RuleGrey, Reason: "captcha",
	})
	if err != nil {
		t.Fatal(err)
	}

	if d := s.VerifyCaptcha(context.Background(), "1.2.3.4", "4"); d.Allowed || d.Reason != models.DenyBlacklisted {
		t.Fatalf("captcha must not bypass deny rule, got allowed=%v reason=%s", d.Allowed, d.Reason)
	}
}
