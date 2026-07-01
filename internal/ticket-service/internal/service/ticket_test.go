package service

import (
	"testing"

	"autorent-backend/internal/ticket-service/internal/models"
)

type fakeTicketStore struct {
	tickets []models.Ticket
}

func (f *fakeTicketStore) Create(ticket *models.Ticket) error {
	f.tickets = append(f.tickets, *ticket)
	return nil
}

func (f *fakeTicketStore) List() ([]models.Ticket, error) {
	return f.tickets, nil
}

func (f *fakeTicketStore) GetByID(id string) (*models.Ticket, error) {
	for i := range f.tickets {
		if f.tickets[i].ID.String() == id {
			return &f.tickets[i], nil
		}
	}
	return nil, nil
}

func (f *fakeTicketStore) Update(ticket *models.Ticket) error {
	for i := range f.tickets {
		if f.tickets[i].ID == ticket.ID {
			f.tickets[i] = *ticket
			return nil
		}
	}
	return nil
}

func (f *fakeTicketStore) Delete(id string) error {
	for i, ticket := range f.tickets {
		if ticket.ID.String() == id {
			f.tickets = append(f.tickets[:i], f.tickets[i+1:]...)
			return nil
		}
	}
	return nil
}

func TestCreateTicket(t *testing.T) {
	repo := &fakeTicketStore{}
	svc := NewTicketService(repo)

	_, err := svc.CreateTicket(struct {
		Type        models.TicketType `json:"type"`
		Title       string            `json:"title"`
		Description string            `json:"description"`
		RequesterID string            `json:"requester_id"`
		ReferenceID string            `json:"reference_id"`
		Metadata    map[string]any    `json:"metadata"`
	}{
		Type:        models.TicketTypeClientOnboarding,
		Description: "test",
		RequesterID: "user-1",
		ReferenceID: "ref-1",
	})
	if err == nil {
		t.Fatalf("expected validation error for empty title")
	}

	ticket, err := svc.CreateTicket(struct {
		Type        models.TicketType `json:"type"`
		Title       string            `json:"title"`
		Description string            `json:"description"`
		RequesterID string            `json:"requester_id"`
		ReferenceID string            `json:"reference_id"`
		Metadata    map[string]any    `json:"metadata"`
	}{
		Type:        models.TicketTypeClientOnboarding,
		Title:       "Need onboarding",
		Description: "test",
		RequesterID: "user-1",
		ReferenceID: "ref-1",
		Metadata: map[string]any{
			"source": "test",
		},
	})
	if err != nil {
		t.Fatalf("expected ticket creation to succeed: %v", err)
	}
	if ticket.Status != models.TicketStatusPending {
		t.Fatalf("expected initial status pending, got %s", ticket.Status)
	}
}
