package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/citaspot/api/internal/domain"
)

type availabilityService struct {
	scheduleRepo domain.ScheduleRepository
	serviceRepo  domain.ServiceRepository
}

// NewAvailabilityService crea el engine de disponibilidad.
func NewAvailabilityService(scheduleRepo domain.ScheduleRepository, serviceRepo domain.ServiceRepository) domain.AvailabilityService {
	return &availabilityService{
		scheduleRepo: scheduleRepo,
		serviceRepo:  serviceRepo,
	}
}

// GetAvailableSlots calcula los slots libres para un profesional + servicio en una fecha.
//
// Algoritmo:
//  1. Cargar el servicio para obtener duration_min + buffer_min.
//  2. Parsear la fecha con el timezone del tenant.
//  3. Obtener el horario semanal del profesional para ese día.
//  4. Generar candidatos cada (duration_min + buffer_min) minutos.
//  5. Eliminar los que se solapan con citas activas o bloqueos.
func (s *availabilityService) GetAvailableSlots(ctx context.Context, tenantID uuid.UUID, query *domain.AvailabilityQuery) ([]*domain.TimeSlot, error) {
	// Timezone por defecto
	tz := query.Timezone
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("%w: timezone inválida '%s'", domain.ErrValidation, tz)
	}

	// 1. Cargar servicio
	svc, err := s.serviceRepo.GetByID(ctx, tenantID, query.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("availabilityService: service: %w", err)
	}
	totalDuration := time.Duration(svc.DurationMin+svc.BufferMin) * time.Minute

	// 2. Parsear fecha en el timezone dado
	day, err := time.ParseInLocation("2006-01-02", query.Date, loc)
	if err != nil {
		return nil, fmt.Errorf("%w: date debe tener formato YYYY-MM-DD", domain.ErrValidation)
	}

	// 0=Dom, 1=Lun … Go: time.Sunday=0, time.Monday=1 — match directo con el schema
	dayOfWeek := int(day.Weekday())

	// 3. Obtener horario del profesional para ese día (puede haber múltiples bloques: ej 08-12 y 14-18)
	schedules, err := s.scheduleRepo.GetSchedules(ctx, tenantID, query.ProfessionalID)
	if err != nil {
		return nil, fmt.Errorf("availabilityService: schedules: %w", err)
	}

	type workBlock struct {
		start, end time.Time
	}
	var workBlocks []workBlock
	for _, sch := range schedules {
		if sch.DayOfWeek == dayOfWeek && sch.IsActive {
			bs, err := parseTimeOnDay(day, sch.StartTime, loc)
			if err != nil {
				return nil, err
			}
			be, err := parseTimeOnDay(day, sch.EndTime, loc)
			if err != nil {
				return nil, err
			}
			workBlocks = append(workBlocks, workBlock{start: bs, end: be})
		}
	}

	if len(workBlocks) == 0 {
		// El profesional no trabaja ese día → retornar lista vacía
		return []*domain.TimeSlot{}, nil
	}

	// Rango del día completo para buscar conflictos.
	// Mantenemos los time.Time en la location del tenant; al serializar a JSON Go
	// usa el offset explícito (-04:00) en lugar de Z. Las comparaciones internas
	// siguen siendo correctas porque dos time.Time con distinta Location pero
	// mismo instante son iguales con Before/After/Equal.
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.Add(24 * time.Hour)

	// 4. Cargar citas existentes y bloqueos en ese rango
	existingAppts, err := s.scheduleRepo.GetAppointmentsInRange(ctx, tenantID, query.ProfessionalID, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("availabilityService: appointments: %w", err)
	}
	blocks, err := s.scheduleRepo.GetBlocks(ctx, tenantID, query.ProfessionalID, dayStart, dayEnd)
	if err != nil {
		return nil, fmt.Errorf("availabilityService: blocks: %w", err)
	}

	// 5. Generar y filtrar slots por cada bloque de trabajo
	now := time.Now().In(loc)
	var slots []*domain.TimeSlot

	for _, wb := range workBlocks {
		slotStart := wb.start
		for {
			slotEnd := slotStart.Add(totalDuration)
			if slotEnd.After(wb.end) {
				break
			}

			// No mostrar slots en el pasado
			if slotStart.Before(now) {
				slotStart = slotStart.Add(totalDuration)
				continue
			}

			if !overlapsAny(slotStart, slotEnd, existingAppts, blocks) {
				slots = append(slots, &domain.TimeSlot{
					StartsAt: slotStart,
					EndsAt:   slotStart.Add(time.Duration(svc.DurationMin) * time.Minute),
				})
			}

			slotStart = slotStart.Add(totalDuration)
		}
	}

	if slots == nil {
		return []*domain.TimeSlot{}, nil
	}
	return slots, nil
}

// parseTimeOnDay combina una fecha con un string "HH:MM" en el timezone dado.
func parseTimeOnDay(day time.Time, hhmm string, loc *time.Location) (time.Time, error) {
	var h, m int
	if _, err := fmt.Sscanf(hhmm, "%d:%d", &h, &m); err != nil {
		return time.Time{}, fmt.Errorf("horario inválido '%s'", hhmm)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, loc), nil
}

// overlapsAny verifica si un slot se solapa con alguna cita o bloqueo.
func overlapsAny(start, end time.Time, appts []*domain.Appointment, blocks []*domain.ScheduleBlock) bool {
	startUTC := start.UTC()
	endUTC := end.UTC()

	for _, a := range appts {
		// Solapamiento: start < a.EndsAt AND end > a.StartsAt
		if startUTC.Before(a.EndsAt) && endUTC.After(a.StartsAt) {
			return true
		}
	}
	for _, b := range blocks {
		if startUTC.Before(b.EndsAt) && endUTC.After(b.StartsAt) {
			return true
		}
	}
	return false
}
