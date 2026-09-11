package user

// Membership is what a tenant's own record says about a subject: the role it holds there,
// and whether that membership is in force.
type Membership struct {
	Role   UserRole
	Status UserStatus
}
