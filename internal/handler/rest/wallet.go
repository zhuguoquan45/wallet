package rest

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zgq/wallet/internal/domain"
	"github.com/zgq/wallet/internal/service"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts wallet routes onto the given gin engine.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/wallets", h.createWallet)
	r.GET("/wallets/:id", h.getWallet)
	r.POST("/wallets/transfer", h.transfer)
	r.POST("/wallets/:id/deposit", h.deposit)
}

func (h *Handler) createWallet(c *gin.Context) {
	wallet, err := h.svc.CreateWallet(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, wallet)
}

func (h *Handler) getWallet(c *gin.Context) {
	wallet, err := h.svc.GetWallet(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, wallet)
}

type transferRequest struct {
	FromID string `json:"from_id"`
	ToID   string `json:"to_id"`
	Amount int64  `json:"amount"`
}

type depositRequest struct {
	Amount int64 `json:"amount"`
}

func (h *Handler) deposit(c *gin.Context) {
	var req depositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	wallet, err := h.svc.Deposit(c.Request.Context(), c.Param("id"), req.Amount)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrInvalidAmount):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, wallet)
}

func (h *Handler) transfer(c *gin.Context) {
	var req transferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	err := h.svc.Transfer(c.Request.Context(), req.FromID, req.ToID, req.Amount)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, domain.ErrInsufficientFunds),
			errors.Is(err, domain.ErrInvalidAmount),
			errors.Is(err, domain.ErrSameWallet):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		default:
			log.Println("transfer error:", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}
