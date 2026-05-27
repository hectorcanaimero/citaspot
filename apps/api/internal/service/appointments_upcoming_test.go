// Tests para ListUpcoming (clamping de limit) y publishRealtime (nil-safe).
// Usan un mock minimal e inline del AppointmentRepository para no depender
// del resto del wiring del service.
package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

// stubApptRepo es un mock minimo de domain.AppointmentRepository que
// captura los argumentos pasados a ListUpcoming para verificar clamping.
type stubApptRepo struct {
	gotLimit    int
	gotStatuses []string
}

func (s *stubApptRepo) Create(ctx context.Context, a *domain.Appointment) error {
	return nil
}
func (s *stubApptRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.AppointmentWithDetails, error) {
	return &domain.AppointmentWithDetails{}, nil
}
func (s *stubApptRepo) ListByDate(ctx context.Context, tenantID uuid.UUID, date, timezone string) ([]*domain.AppointmentWithDetails, error) {
	return nil, nil
}
func (s *stubApptRepo) ListFiltered(ctx context.Context, tenantID uuid.UUID, q *domain.AppointmentListQuery) (*domain.PaginatedAppointments, error) {
	return nil, nil
}
func (s *stubApptRepo) ListUpcomingByCustomer(ctx context.Context, tenantID, customerID uuid.UUID) ([]*domain.AppointmentWithDetails, error) {
	return nil, nil
}
func (s *stubApptRepo) ListUpcoming(ctx context.Context, tenantID uuid.UUID, limit int, statuses []string) ([]*domain.AppointmentWithDetails, error) {
	s.gotLimit = limit
	s.gotStatuses = statuses
	return []*domain.AppointmentWithDetails{}, nil
}
func (s *stubApptRepo) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, req *domain.UpdateAppointmentRequest) error {
	return nil
}
func (s *stubApptRepo) CheckConflict(ctx context.Context, tenantID, professionalID uuid.UUID, startsAt, endsAt time.Time, excludeID *uuid.UUID) (bool, error) {
	return false, nil
}
func (s *stubApptRepo) Reschedule(ctx context.Context, tenantID, id, professionalID uuid.UUID, startsAt, endsAt time.Time) error {
	return nil
}
func (s *stubApptRepo) ListDistinctCustomersByProfessional(ctx context.Context, tenantID, professionalID uuid.UUID) ([]*domain.Customer, error) {
	return nil, nil
}

// newApptSvcWithRepo construye un appointmentSvc apuntando al stub. Solo
// el campo apptRepo se usa en ListUpcoming, los demas quedan nil.
func newApptSvcWithRepo(repo domain.AppointmentRepository) *appointmentSvc {
	return &appointmentSvc{apptRepo: repo}
}

func TestListUpcoming_ClampsLimitTo50(t *testing.T) {
	repo := &stubApptRepo{}
	svc := newApptSvcWithRepo(repo)

	_, err := svc.ListUpcoming(context.Background(), uuid.New(), 999)
	if err != nil {
		t.Fatalf("ListUpcoming: %v", err)
	}
	if repo.gotLimit != 50 {
		t.Errorf("limit clampado mal: esperado 50, got %d", repo.gotLimit)
	}
	if len(repo.gotStatuses) != 2 || repo.gotStatuses[0] != "pending" || repo.gotStatuses[1] != "confirmed" {
		t.Errorf("statuses esperados [pending confirmed], got %v", repo.gotStatuses)
	}
}

func TestListUpcoming_NegativeBecomesDefault10(t *testing.T) {
	repo := &stubApptRepo{}
	svc := newApptSvcWithRepo(repo)

	_, err := svc.ListUpcoming(context.Background(), uuid.New(), -5)
	if err != nil {
		t.Fatalf("ListUpcoming: %v", err)
	}
	if repo.gotLimit != 10 {
		t.Errorf("limit default esperado 10, got %d", repo.gotLimit)
	}
}

func TestListUpcoming_ZeroBecomesDefault10(t *testing.T) {
	repo := &stubApptRepo{}
	svc := newApptSvcWithRepo(repo)

	_, err := svc.ListUpcoming(context.Background(), uuid.New(), 0)
	if err != nil {
		t.Fatalf("ListUpcoming: %v", err)
	}
	if repo.gotLimit != 10 {
		t.Errorf("limit default esperado 10 para 0, got %d", repo.gotLimit)
	}
}

func TestListUpcoming_ValidLimitPasses(t *testing.T) {
	repo := &stubApptRepo{}
	svc := newApptSvcWithRepo(repo)

	_, err := svc.ListUpcoming(context.Background(), uuid.New(), 25)
	if err != nil {
		t.Fatalf("ListUpcoming: %v", err)
	}
	if repo.gotLimit != 25 {
		t.Errorf("limit pass-through fallo: esperado 25, got %d", repo.gotLimit)
	}
}

// TestPublishRealtime_NilRdbIsNoop verifica que publishRealtime no panic ni
// retorna error cuando rdb es nil (modo degradado sin Redis al startup).
func TestPublishRealtime_NilRdbIsNoop(t *testing.T) {
	// Service con rdb=nil — caso degradado.
	svc := &appointmentSvc{rdb: nil}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("publishRealtime panic con rdb nil: %v", r)
		}
	}()

	// No debe panic ni necesitar contactar el apptRepo (porque retorna early).
	svc.publishRealtime(context.Background(), uuid.New(), uuid.New(), "appointment.created")
}
