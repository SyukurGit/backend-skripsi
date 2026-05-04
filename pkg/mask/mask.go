package mask

import "unicode/utf8"

func Phone(phone string) string {
	// VALIDASI: masking data untuk mencegah kebocoran informasi sensitif
	// Target: 0812****123 (mask tengah).
	if phone == "" {
		return ""
	}
	if utf8.RuneCountInString(phone) < 7 {
		return "****"
	}

	r := []rune(phone)
	start := 4
	end := len(r) - 3
	if end <= start {
		return "****"
	}

	for i := start; i < end; i++ {
		r[i] = '*'
	}
	return string(r)
}
