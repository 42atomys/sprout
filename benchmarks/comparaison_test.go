package benchmarks_test

import (
	"bytes"
	"log/slog"
	"testing"
	"text/template"

	"github.com/42atomys/sprout"
	"github.com/Masterminds/sprig/v3"
)

/**
 * BenchmarkSprig are the benchmarks for Sprig.
 * It is the same as SproutBench but with Sprig.
 */
func BenchmarkSprig(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tmpl, err := template.New("allFunctions").Funcs(sprig.FuncMap()).ParseGlob("*.sprig.tmpl")
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		for _, t := range tmpl.Templates() {
			err := tmpl.ExecuteTemplate(&buf, t.Name(), nil)
			if err != nil {
				panic(err)
			}
		}

		buf.Reset()
	}
}

/**
 * BenchmarkSprout are the benchmarks for Sprout.
 */
func BenchmarkSprout(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		errChan := make(chan error)
		defer close(errChan)

		fnHandler := sprout.NewFunctionHandler(
			sprout.WithErrStrategy(sprout.ErrorStrategyPanic),
			sprout.WithLogger(slog.New(&slog.TextHandler{})),
			sprout.WithErrorChannel(errChan),
		)

		go func() {
			for err := range errChan {
				fnHandler.Logger().Error(err.Error())
			}
		}()

		tmpl, err := template.New("allFunctions").Funcs(sprout.FuncMap(sprout.WithFunctionHandler(fnHandler))).ParseGlob("*.sprout.tmpl")

		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		for _, t := range tmpl.Templates() {
			err := tmpl.ExecuteTemplate(&buf, t.Name(), fnHandler)
			if err != nil {
				panic(err)
			}
		}
		buf.Reset()
	}
}
