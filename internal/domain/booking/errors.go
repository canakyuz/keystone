package booking

import "errors"

var (
	// Availability errors
	ErrAvailabilityNotFound      = errors.New("availability not found")
	ErrAvailabilityAlreadyBooked = errors.New("availability already booked")
	ErrAvailabilityConflict      = errors.New("availability time slot conflict")
	ErrInvalidAvailabilityTime   = errors.New("invalid availability time range")

	// Appointment errors
	ErrAppointmentNotFound   = errors.New("appointment not found")
	ErrAppointmentCancelled  = errors.New("appointment is cancelled")
	ErrAppointmentCompleted  = errors.New("appointment is already completed")
	ErrAppointmentConflict   = errors.New("appointment time slot conflict")
	ErrInvalidAppointmentTime = errors.New("invalid appointment time range")
	ErrAppointmentInPast     = errors.New("cannot create appointment in the past")
)
