package resources

import (
	"github.com/pergus/api-server/pkg/api"
)

// User is a sample resource type.
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	IsActive bool   `json:"is_active"`
}

// UserResource implements the Resource interface.
type UserResource struct {
	storage api.Storage
}

// NewUserResource creates a new user resource.
func NewUserResource() *UserResource {
	return &UserResource{
		storage: api.NewMemoryStorage(),
	}
}

// Name returns "users".
func (r *UserResource) Name() string {
	return "users"
}

// NewObject returns an empty User.
func (r *UserResource) NewObject() any {
	return &User{}
}

// Storage returns the storage implementation.
func (r *UserResource) Storage() api.Storage {
	return r.storage
}
