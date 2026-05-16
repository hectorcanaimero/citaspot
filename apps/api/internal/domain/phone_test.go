package domain

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "digitos sin prefijo", input: "4243148415", want: "+4243148415"},
		{name: "con prefijo +", input: "+584243148415", want: "+584243148415"},
		{name: "con guiones", input: "+58-424-314-8415", want: "+584243148415"},
		{name: "con espacios", input: "+58 424 314 8415", want: "+584243148415"},
		{name: "con parentesis", input: "(809) 555-1234", want: "+8095551234"},
		{name: "prefijo sin +", input: "584243148415", want: "+584243148415"},
		{name: "string vacio", input: "", want: ""},
		{name: "solo caracteres invalidos", input: "abc", want: ""},
		{name: "formato WA normalizado", input: "+18091234567", want: "+18091234567"},
		{name: "con puntos", input: "58.424.314.8415", want: "+584243148415"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePhone(tt.input)
			if got != tt.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
