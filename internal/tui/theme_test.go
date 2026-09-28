package tui_test

import (
	"image/color"
	"reflect"
	"testing"

	"github.com/Lakelimbo/nezumi/internal/tui"
)

func TestDefaultThemeIsFullyPopulated(t *testing.T) {
	th := tui.DefaultTheme()

	value := reflect.ValueOf(th)
	typ := value.Type()

	for i := range typ.NumField() {
		field := typ.Field(i)

		c, ok := value.Field(i).Interface().(color.Color)
		if !ok {
			t.Fatalf("Theme.%s is %s, want a color.Color", field.Name, field.Type)
		}

		if c == nil {
			t.Errorf("Theme.%s is nil: styles built from it render unstyled", field.Name)
		}
	}
}
