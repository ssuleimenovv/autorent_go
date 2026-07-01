package service

import (
	"encoding/json"
	"fmt"

	"autorent-backend/internal/ticket-service/internal/models"
	"github.com/google/uuid"
)

type TicketStore interface {
	Create(ticket *models.Ticket) error
	List() ([]models.Ticket, error)
	GetByID(id string) (*models.Ticket, error)
	Update(ticket *models.Ticket) error
	Delete(id string) error
}

type TicketService struct {
	repo TicketStore
}

func NewTicketService(repo TicketStore) *TicketService {
	return &TicketService{repo: repo}
}

func (s *TicketService) CreateTicket(input struct {
	Type        models.TicketType `json:"type"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	RequesterID string            `json:"requester_id"`
	ReferenceID string            `json:"reference_id"`
	Metadata    map[string]any    `json:"metadata"`
}) (*models.Ticket, error) {
	if input.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	if input.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	if input.RequesterID == "" {
		return nil, fmt.Errorf("requester_id is required")
	}
	if input.ReferenceID == "" {
		return nil, fmt.Errorf("reference_id is required")
	}

	metadataBytes, _ := json.Marshal(input.Metadata)
	ticket := &models.Ticket{
		ID:          uuid.New(),
		Type:        input.Type,
		Status:      models.TicketStatusPending,
		Title:       input.Title,
		Description: input.Description,
		RequesterID: input.RequesterID,
		ReferenceID: input.ReferenceID,
		Metadata:    string(metadataBytes),
	}

	if err := s.repo.Create(ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *TicketService) ListTickets() ([]models.Ticket, error) {
	return s.repo.List()
}

func (s *TicketService) GetTicket(id string) (*models.Ticket, error) {
	return s.repo.GetByID(id)
}

func (s *TicketService) UpdateTicketStatus(id string, status models.TicketStatus) (*models.Ticket, error) {
	ticket, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	ticket.Status = status
	if err := s.repo.Update(ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}
