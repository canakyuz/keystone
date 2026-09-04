package user

import "fmt"

// Bu dosya, domain entity'sindeki opsiyonel alanlar ile Postgres'in nullable
// kolonları arasındaki dönüşümü tek yerde toplar.
//
// NEDEN gerekli: user.User, Avatar / Phone / Timezone / Locale / CreatedBy / UpdatedBy
// alanlarını düz string olarak tutuyor. Postgres tarafında bu kolonlar nullable
// ve ikisi UUID tipinde. Düz string'i doğrudan geçirmek iki hataya yol açıyordu:
//
//   - Yazarken: boş string UUID kolonuna gidiyor ve
//     `invalid input syntax for type uuid: ""` hatası veriyordu.
//   - Okurken: NULL kolon *string hedefine taranıyor ve
//     `converting NULL to string is unsupported` hatası veriyordu.
//
// Alternatif, entity alanlarını *string yapmaktı. Bunu tercih etmedik çünkü
// domain katmanını veritabanı nullability'sine göre şekillendirmek katman
// bağımlılığını ters çevirir. Dönüşüm repository sınırında kalmalı.

// nullable, boş string'i SQL NULL'a çevirir; dolu değeri olduğu gibi geçirir.
// Karmaşıklık: O(1).
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullString, NULL okunabilen bir sql.Scanner hedefidir. NULL geldiğinde
// hedefi boş string'e ayarlar, böylece entity'nin "değer yok" temsili korunur.
type nullString struct {
	dst *string
}

// Scan, sql.Scanner arayüzünü karşılar.
// Karmaşıklık: O(n), n = değerin bayt uzunluğu.
func (n nullString) Scan(src any) error {
	if src == nil {
		*n.dst = ""
		return nil
	}

	switch v := src.(type) {
	case string:
		*n.dst = v
	case []byte:
		*n.dst = string(v)
	default:
		return fmt.Errorf("nullString: desteklenmeyen kaynak tipi %T", src)
	}

	return nil
}
