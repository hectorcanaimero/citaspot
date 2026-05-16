package domain

import "strings"

// NormalizePhone limpia un numero de telefono: elimina caracteres no numericos
// excepto el prefijo "+", y asegura que siempre comience con "+".
// Ejemplos:
//
//	"4243148415"       -> "+4243148415"
//	"+58-424-314-8415" -> "+584243148415"
//	"584243148415"     -> "+584243148415"
//	""                 -> ""
func NormalizePhone(phone string) string {
	if phone == "" {
		return ""
	}

	hasPlus := strings.HasPrefix(phone, "+")

	// Eliminar todo excepto digitos
	var sb strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}

	digits := sb.String()
	if digits == "" {
		return ""
	}

	if hasPlus {
		return "+" + digits
	}
	return "+" + digits
}
