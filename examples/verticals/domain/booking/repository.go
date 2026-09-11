package booking

import "context"

// AvailabilityRepository defines availability data access interface
type AvailabilityRepository interface {
	Create(ctx context.Context, availability *Availability) error
	GetByID(ctx context.Context, id string) (*Availability, error)
	List(ctx context.Context, filters AvailabilityListFilters) ([]*Availability, int64, error)
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Availability, int64, error)
	GetByDateRange(ctx context.Context, startDate, endDate string) ([]*Availability, error)
	Update(ctx context.Context, availability *Availability) error
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context) (map[string]any, error)
}

// AppointmentRepository defines appointment data access interface
type AppointmentRepository interface {
	Create(ctx context.Context, appointment *Appointment) error
	GetByID(ctx context.Context, id string) (*Appointment, error)
	List(ctx context.Context, filters AppointmentListFilters) ([]*Appointment, int64, error)
	GetByUser(ctx context.Context, userID string, limit, offset int) ([]*Appointment, int64, error)
	GetByClient(ctx context.Context, clientEmail string, limit, offset int) ([]*Appointment, int64, error)
	GetUpcoming(ctx context.Context, limit int) ([]*Appointment, error)
	GetByDateRange(ctx context.Context, startDate, endDate string) ([]*Appointment, error)
	Update(ctx context.Context, appointment *Appointment) error
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context) (map[string]any, error)
}

// AvailabilityListFilters defines filters for listing availabilities
type AvailabilityListFilters struct {
	UserID    *string
	Status    *AvailabilityStatus
	StartDate *string
	EndDate   *string
	Search    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}

// AppointmentListFilters defines filters for listing appointments
type AppointmentListFilters struct {
	UserID      *string
	ClientEmail *string
	Status      *AppointmentStatus
	Type        *AppointmentType
	StartDate   *string
	EndDate     *string
	Search      string
	SortBy      string
	SortOrder   string
	Limit       int
	Offset      int
}
