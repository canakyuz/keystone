// Package operation, uzun suren islemlerin kalici takibini modeller.
//
// Iki kavram ayri tutulur:
//
//   - Operation: kullaniciya donuk kayit. "Acme'nin kurulumu ne durumda?"
//   - Job: calistirana donuk is birimi. "Bu isi kim, ne zamana kadar aldi?"
//
// Ayrimin nedeni: bir tenant'in durumu ile tek bir islemin durumu ayni sey
// degildir. Basarisiz bir kurulumdan sonra ayni tenant icin yeni bir operasyon
// acilabilir. Ikisini tek alanda tutmak bu ayrimi imkansiz kilardi.
package operation

import (
	"errors"
	"fmt"
	"time"
)

// Kind, islem turudur.
type Kind string

const (
	// KindTenantProvision, yeni bir tenant icin calisma alani hazirlar.
	KindTenantProvision Kind = "tenant.provision"
)

// Status, operasyonun durumudur.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

// IsTerminal, durumun daha fazla degismeyecegini soyler.
func (s Status) IsTerminal() bool {
	return s == StatusSucceeded || s == StatusFailed
}

var (
	// ErrNotFound, operasyonun bulunamadigini bildirir.
	ErrNotFound = errors.New("operation: bulunamadı")

	// ErrInvalidTransition, izin verilmeyen bir durum gecisini bildirir.
	ErrInvalidTransition = errors.New("operation: geçersiz durum geçişi")

	// ErrStaleFence, gecikmis bir worker bildirimini reddeder.
	ErrStaleFence = errors.New("operation: fence eskimiş, bildirim reddedildi")

	// ErrIdempotencyConflict, ayni anahtarin farkli govdeyle kullanildigini
	// bildirir.
	ErrIdempotencyConflict = errors.New("operation: idempotency anahtarı farklı istekle kullanılmış")
)

// Operation, kullaniciya donuk islem kaydidir.
type Operation struct {
	ID       string
	TenantID string
	Kind     Kind
	Status   Status

	ErrorCode    string
	ErrorMessage string

	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// allowedTransitions, izin verilen durum geciseridir.
//
// Tabloyu acikca yazmak, gecerli gecisleri kodun icine dagilmis if'lerden
// okumaya calismaktan iyidir: hangi gecisin neden yasak oldugu tek yerde
// gorulur ve yeni bir durum eklendiginde unutulmaz.
var allowedTransitions = map[Status][]Status{
	StatusPending:   {StatusRunning, StatusFailed},
	StatusRunning:   {StatusSucceeded, StatusFailed},
	StatusSucceeded: {},
	StatusFailed:    {},
}

// CanTransitionTo, gecisin izinli olup olmadigini soyler.
// Karmasiklik: O(k), k = bir durumdan cikan gecis sayisi (en fazla 2).
func (o *Operation) CanTransitionTo(next Status) bool {
	for _, allowed := range allowedTransitions[o.Status] {
		if allowed == next {
			return true
		}
	}
	return false
}

// MarkRunning, operasyonu calisiyor olarak isaretler.
func (o *Operation) MarkRunning() error {
	return o.transition(StatusRunning, nil)
}

// MarkSucceeded, operasyonu basarili olarak kapatir.
func (o *Operation) MarkSucceeded(now time.Time) error {
	return o.transition(StatusSucceeded, &now)
}

// MarkFailed, operasyonu hatayla kapatir.
//
// Hata kodu zorunludur: istemci koda gore dallanir. Yalnizca serbest metin
// birakmak, cagiranin mesaj icerigine gore dallanmasina yol acar.
func (o *Operation) MarkFailed(code, message string, now time.Time) error {
	if code == "" {
		return fmt.Errorf("%w: hata kodu zorunlu", ErrInvalidTransition)
	}

	if err := o.transition(StatusFailed, &now); err != nil {
		return err
	}

	o.ErrorCode = code
	o.ErrorMessage = message

	return nil
}

// transition, durum degisimini dogrular ve uygular.
func (o *Operation) transition(next Status, completedAt *time.Time) error {
	if !o.CanTransitionTo(next) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, o.Status, next)
	}

	o.Status = next
	o.CompletedAt = completedAt

	return nil
}
