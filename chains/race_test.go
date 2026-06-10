// SPDX-License-Identifier: MIT

package chains

import (
	"sync"
	"testing"

	"github.com/Emin-ACIKGOZ/go-skrub/pkg/core"
)

func TestStringRule_ConcurrentRace(t *testing.T) {
	val := "hello world"
	config := CompileStringConfig([]func(*StringChain){
		func(c *StringChain) { c.Min(3) },
		func(c *StringChain) { c.Max(100) },
	})
	rule := NewStringRule(config, &val, "val")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := core.NewContext(core.Config{})
			if err := rule.Validate(ctx); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestIntRule_ConcurrentRace(t *testing.T) {
	val := 42
	config := CompileIntConfig([]func(*IntChain){
		func(c *IntChain) { c.Min(10) },
		func(c *IntChain) { c.Max(100) },
	})
	rule := NewIntRule(config, &val, "val")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := core.NewContext(core.Config{})
			if err := rule.Validate(ctx); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestSliceRule_ConcurrentRace(t *testing.T) {
	data := make([]string, 50)
	for i := range data {
		data[i] = "hello"
	}

	// Build element template manually using a StringChain and compile
	elemConfig := CompileStringConfig([]func(*StringChain){
		func(c *StringChain) { c.Min(2); c.Max(10) },
	})
	elemTpl := &simpleStringTemplate{config: elemConfig}

	config := CompileSliceConfig([]func(*SliceChain){
		func(c *SliceChain) { c.MinLen(1) },
	})
	rule := NewSliceRule(config, &data, "data", []core.Template{elemTpl})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := core.NewContext(core.Config{})
			if err := rule.Validate(ctx); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestStringRule_NilCtx(t *testing.T) {
	val := "hello"
	config := CompileStringConfig([]func(*StringChain){
		func(c *StringChain) { c.Min(1) },
	})
	rule := NewStringRule(config, &val, "val")
	if err := rule.Validate(nil); err != nil {
		t.Errorf("expected nil ctx to work, got: %v", err)
	}
}

// simpleStringTemplate wraps a ChainConfig as a Template for testing.
type simpleStringTemplate struct {
	config *core.ChainConfig
}

func (s *simpleStringTemplate) Bind(target any, name string) core.Rule {
	return NewStringRule(s.config, target, name)
}

func TestAccumulateMode_ConcurrentRace(_ *testing.T) {
	val := "hello"
	config := CompileStringConfig([]func(*StringChain){
		func(c *StringChain) { c.Min(3) },
	})
	rule := NewStringRule(config, &val, "val")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := core.NewContext(core.Config{AccumulateErrors: true})
			_ = rule.Validate(ctx)
		}()
	}
	wg.Wait()
}
