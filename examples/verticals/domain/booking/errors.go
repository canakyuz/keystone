package booking

import "github.com/canakyuz/keystone/pkg/clienterr"

var (
	// Availability errors
	ErrAvailabilityNotFound      = clienterr.New("availability not found")
	ErrAvailabilityAlreadyBooked = clienterr.New("availability already booked")
	ErrAvailabilityConflict      = clienterr.New("availability time slot conflict")
	ErrInvalidAvailabilityTime   = clienterr.New("invalid availability time range")

	// Appointment errors
	ErrAppointmentNotFound    = clienterr.New("appointment not found")
	ErrAppointmentCancelled   = clienterr.New("appointment is cancelled")
	ErrAppointmentCompleted   = clienterr.New("appointment is already completed")
	ErrAppointmentConflict    = clienterr.New("appointment time slot conflict")
	ErrInvalidAppointmentTime = clienterr.New("invalid appointment time range")
	ErrAppointmentInPast      = clienterr.New("cannot create appointment in the past")
)
