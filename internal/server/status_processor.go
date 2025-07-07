package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/and161185/loyalty/internal/model"
)

func (s *Server) OrdersStatusControl(ctx context.Context) {
	workerCount := 5

	ch := make(chan model.Order, 10*workerCount)
	go s.ProcessOrders(ctx, ch)

	for i := 0; i < workerCount; i++ {
		go s.UpdateOrder(ctx, ch)
	}
}

func (s *Server) ProcessOrders(ctx context.Context, ch chan model.Order) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			orders, err := s.orderStorage.GetUnprocessedOrders(ctx)
			if err != nil {
				s.deps.Logger.Errorf("process orders: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			skipped := 0
			for _, order := range orders {
				select {
				case ch <- order:

				default:
					skipped++
					if skipped%10 == 0 {
						s.deps.Logger.Warnf("channel full, skipped %d orders", skipped)
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *Server) UpdateOrder(ctx context.Context, ch chan model.Order) {
	for {
		select {
		case <-ctx.Done():
			return
		case order := <-ch:
			newStatusOrder, err := s.getStatus(ctx, order)
			if err != nil {
				s.deps.Logger.Errorf("get order status: %v", err)
				continue
			}
			if newStatusOrder.Status == order.Status {
				continue
			}
			err = s.orderStorage.UpdateOrder(ctx, newStatusOrder)
			if err != nil {
				s.deps.Logger.Errorf("update order: %v", err)
			}
		}

	}
}

func (s *Server) getStatus(ctx context.Context, order model.Order) (model.Order, error) {
	url := fmt.Sprintf("%s/api/orders/%s", s.config.AccrualSystemAddress, order.Number)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return order, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return order, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return order, nil
	case http.StatusTooManyRequests:
		if retry := resp.Header.Get("Retry-After"); retry != "" {
			if sec, err := strconv.Atoi(retry); err == nil {
				time.Sleep(time.Duration(sec) * time.Second)
			}
		}
		return order, fmt.Errorf("too many requests")
	case http.StatusOK:
		var response struct {
			Order   string   `json:"order"`
			Status  string   `json:"status"`
			Accrual *float64 `json:"accrual,omitempty"`
		}

		err := json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return order, fmt.Errorf("decode response: %w", err)
		}

		order.Status = model.OrderStatus(response.Status)
		order.Accrual = response.Accrual
		return order, nil

	default:
		return order, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
