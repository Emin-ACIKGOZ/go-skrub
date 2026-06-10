// SPDX-License-Identifier: MIT

package chains

import (
	"regexp"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

// crossValidationCase runs the same validation against both an old chain
// and a new rule, asserting they produce identical results.
type crossValidationCase struct {
	name     string
	oldRule  core.Rule
	newRule  core.Rule
	wantFail bool
}

func runCrossValidation(t *testing.T, tc crossValidationCase) {
	t.Helper()

	oldCtx := core.NewContext(core.Config{})
	newCtx := core.NewContext(core.Config{})

	oldErr := tc.oldRule.Validate(oldCtx)
	newErr := tc.newRule.Validate(newCtx)

	if (oldErr != nil) != tc.wantFail {
		t.Errorf("old chain: wantFail=%v, got err=%v", tc.wantFail, oldErr)
	}
	if (newErr != nil) != tc.wantFail {
		t.Errorf("new rule: wantFail=%v, got err=%v", tc.wantFail, newErr)
	}

	if tc.wantFail && oldErr != nil && newErr != nil {
		if oldErr.Error() != newErr.Error() {
			t.Errorf("error message mismatch:\nold: %v\nnew: %v", oldErr, newErr)
		}
	}
}

func TestCrossValidation_StringPass(t *testing.T) {
	val := "hello world"

	oldRule := NewStringChain(&val, "val")
	oldRule.Min(3).Max(100)

	config := CompileStringConfig([]func(*StringChain){
		func(c *StringChain) { c.Min(3); c.Max(100) },
	})
	newRule := NewStringRule(config, &val, "val")

	runCrossValidation(t, crossValidationCase{
		name:     "StringPass",
		oldRule:  oldRule,
		newRule:  newRule,
		wantFail: false,
	})
}

func TestCrossValidation_StringFailMin(t *testing.T) {
	val := "ab"

	oldRule := NewStringChain(&val, "val").Min(3)
	config := CompileStringConfig([]func(*StringChain){
		func(c *StringChain) { c.Min(3) },
	})
	newRule := NewStringRule(config, &val, "val")

	runCrossValidation(t, crossValidationCase{
		name:     "StringFailMin",
		oldRule:  oldRule,
		newRule:  newRule,
		wantFail: true,
	})
}

func TestCrossValidation_StringFailMax(t *testing.T) {
	val := "hello world!!!"
	oldRule := NewStringChain(&val, "val").Max(5)
	config := CompileStringConfig([]func(*StringChain){
		func(c *StringChain) { c.Max(5) },
	})
	newRule := NewStringRule(config, &val, "val")

	runCrossValidation(t, crossValidationCase{
		name:     "StringFailMax",
		oldRule:  oldRule,
		newRule:  newRule,
		wantFail: true,
	})
}

func TestCrossValidation_StringEmail(t *testing.T) {
	valid := "user@example.com"
	invalid := "not-an-email"

	t.Run("Valid", func(t *testing.T) {
		oldRule := NewStringChain(&valid, "email").Email()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.Email() },
		})
		newRule := NewStringRule(config, &valid, "email")
		runCrossValidation(t, crossValidationCase{
			name: "EmailValid", oldRule: oldRule, newRule: newRule, wantFail: false,
		})
	})

	t.Run("Invalid", func(t *testing.T) {
		oldRule := NewStringChain(&invalid, "email").Email()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.Email() },
		})
		newRule := NewStringRule(config, &invalid, "email")
		runCrossValidation(t, crossValidationCase{
			name: "EmailInvalid", oldRule: oldRule, newRule: newRule, wantFail: true,
		})
	})
}

func TestCrossValidation_StringURL(t *testing.T) {
	valid := "https://example.com"
	invalid := "not-a-url"

	t.Run("Valid", func(t *testing.T) {
		oldRule := NewStringChain(&valid, "url").URL()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.URL() },
		})
		newRule := NewStringRule(config, &valid, "url")
		runCrossValidation(t, crossValidationCase{
			name: "URLValid", oldRule: oldRule, newRule: newRule, wantFail: false,
		})
	})

	t.Run("Invalid", func(t *testing.T) {
		oldRule := NewStringChain(&invalid, "url").URL()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.URL() },
		})
		newRule := NewStringRule(config, &invalid, "url")
		runCrossValidation(t, crossValidationCase{
			name: "URLInvalid", oldRule: oldRule, newRule: newRule, wantFail: true,
		})
	})
}

func TestCrossValidation_IntPass(t *testing.T) {
	val := 42
	oldRule := NewIntChain(&val, "val").Min(10).Max(100)
	config := CompileIntConfig([]func(*IntChain){
		func(c *IntChain) { c.Min(10); c.Max(100) },
	})
	newRule := NewIntRule(config, &val, "val")

	runCrossValidation(t, crossValidationCase{
		name: "IntPass", oldRule: oldRule, newRule: newRule, wantFail: false,
	})
}

func TestCrossValidation_IntFailMin(t *testing.T) {
	val := 5
	oldRule := NewIntChain(&val, "val").Min(10)
	config := CompileIntConfig([]func(*IntChain){
		func(c *IntChain) { c.Min(10) },
	})
	newRule := NewIntRule(config, &val, "val")

	runCrossValidation(t, crossValidationCase{
		name: "IntFailMin", oldRule: oldRule, newRule: newRule, wantFail: true,
	})
}

func TestCrossValidation_IntFailMax(t *testing.T) {
	val := 200
	oldRule := NewIntChain(&val, "val").Max(100)
	config := CompileIntConfig([]func(*IntChain){
		func(c *IntChain) { c.Max(100) },
	})
	newRule := NewIntRule(config, &val, "val")

	runCrossValidation(t, crossValidationCase{
		name: "IntFailMax", oldRule: oldRule, newRule: newRule, wantFail: true,
	})
}

func TestCrossValidation_IntNotZero(t *testing.T) {
	val := 0
	oldRule := NewIntChain(&val, "val").NotZero()
	config := CompileIntConfig([]func(*IntChain){
		func(c *IntChain) { c.NotZero() },
	})
	newRule := NewIntRule(config, &val, "val")

	runCrossValidation(t, crossValidationCase{
		name: "IntNotZero", oldRule: oldRule, newRule: newRule, wantFail: true,
	})
}

func TestCrossValidation_StringNotEmpty(t *testing.T) {
	val := ""
	oldRule := NewStringChain(&val, "val").NotEmpty()
	config := CompileStringConfig([]func(*StringChain){
		func(c *StringChain) { c.NotEmpty() },
	})
	newRule := NewStringRule(config, &val, "val")

	runCrossValidation(t, crossValidationCase{
		name: "StringNotEmpty", oldRule: oldRule, newRule: newRule, wantFail: true,
	})
}

func TestCrossValidation_StringUUID(t *testing.T) {
	valid := "550e8400-e29b-41d4-a716-446655440000"
	invalid := "not-a-uuid"

	t.Run("Valid", func(t *testing.T) {
		oldRule := NewStringChain(&valid, "uuid").UUID()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.UUID() },
		})
		newRule := NewStringRule(config, &valid, "uuid")
		runCrossValidation(t, crossValidationCase{
			name: "UUIDValid", oldRule: oldRule, newRule: newRule, wantFail: false,
		})
	})

	t.Run("Invalid", func(t *testing.T) {
		oldRule := NewStringChain(&invalid, "uuid").UUID()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.UUID() },
		})
		newRule := NewStringRule(config, &invalid, "uuid")
		runCrossValidation(t, crossValidationCase{
			name: "UUIDInvalid", oldRule: oldRule, newRule: newRule, wantFail: true,
		})
	})
}

//nolint:goconst // must be variables for &addr
func TestCrossValidation_StringIP(t *testing.T) {
	v4 := "192.168.1.1"
	v6 := "2001:db8::1"
	invalid := "999.999.999.999"

	t.Run("IPv4", func(t *testing.T) {
		oldRule := NewStringChain(&v4, "ip").IP()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.IP() },
		})
		newRule := NewStringRule(config, &v4, "ip")
		runCrossValidation(t, crossValidationCase{
			name: "IPv4", oldRule: oldRule, newRule: newRule, wantFail: false,
		})
	})

	t.Run("IPv6", func(t *testing.T) {
		oldRule := NewStringChain(&v6, "ip").IP()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.IP() },
		})
		newRule := NewStringRule(config, &v6, "ip")
		runCrossValidation(t, crossValidationCase{
			name: "IPv6", oldRule: oldRule, newRule: newRule, wantFail: false,
		})
	})

	t.Run("Invalid", func(t *testing.T) {
		oldRule := NewStringChain(&invalid, "ip").IP()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.IP() },
		})
		newRule := NewStringRule(config, &invalid, "ip")
		runCrossValidation(t, crossValidationCase{
			name: "Invalid", oldRule: oldRule, newRule: newRule, wantFail: true,
		})
	})
}

func TestCrossValidation_StringIPv4(t *testing.T) {
	v4 := "192.168.1.1"
	v6 := "2001:db8::1"

	t.Run("Valid", func(t *testing.T) {
		oldRule := NewStringChain(&v4, "ipv4").IPv4()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.IPv4() },
		})
		newRule := NewStringRule(config, &v4, "ipv4")
		runCrossValidation(t, crossValidationCase{
			name: "Valid", oldRule: oldRule, newRule: newRule, wantFail: false,
		})
	})

	t.Run("RejectIPv6", func(t *testing.T) {
		oldRule := NewStringChain(&v6, "ipv4").IPv4()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.IPv4() },
		})
		newRule := NewStringRule(config, &v6, "ipv4")
		runCrossValidation(t, crossValidationCase{
			name: "RejectIPv6", oldRule: oldRule, newRule: newRule, wantFail: true,
		})
	})
}

func TestCrossValidation_StringIPv6(t *testing.T) {
	v4 := "192.168.1.1"
	v6 := "2001:db8::1"

	t.Run("Valid", func(t *testing.T) {
		oldRule := NewStringChain(&v6, "ipv6").IPv6()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.IPv6() },
		})
		newRule := NewStringRule(config, &v6, "ipv6")
		runCrossValidation(t, crossValidationCase{
			name: "Valid", oldRule: oldRule, newRule: newRule, wantFail: false,
		})
	})

	t.Run("RejectIPv4", func(t *testing.T) {
		oldRule := NewStringChain(&v4, "ipv6").IPv6()
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.IPv6() },
		})
		newRule := NewStringRule(config, &v4, "ipv6")
		runCrossValidation(t, crossValidationCase{
			name: "RejectIPv4", oldRule: oldRule, newRule: newRule, wantFail: true,
		})
	})
}

//nolint:goconst // test strings must be variables for &ref to work
func TestCrossValidation_StringPattern(t *testing.T) {
	re := regexp.MustCompile(`^[a-z]+$`)
	pass := "hello"
	fail := "Hello123"

	t.Run("Pass", func(t *testing.T) {
		oldRule := NewStringChain(&pass, "val").Pattern(re)
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.Pattern(re) },
		})
		newRule := NewStringRule(config, &pass, "val")
		runCrossValidation(t, crossValidationCase{
			name: "Pass", oldRule: oldRule, newRule: newRule, wantFail: false,
		})
	})

	t.Run("Fail", func(t *testing.T) {
		oldRule := NewStringChain(&fail, "val").Pattern(re)
		config := CompileStringConfig([]func(*StringChain){
			func(c *StringChain) { c.Pattern(re) },
		})
		newRule := NewStringRule(config, &fail, "val")
		runCrossValidation(t, crossValidationCase{
			name: "Fail", oldRule: oldRule, newRule: newRule, wantFail: true,
		})
	})
}
