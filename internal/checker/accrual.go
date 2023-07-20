package checker

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AntonPashechko/ya-diplom/internal/config"
	"github.com/AntonPashechko/ya-diplom/internal/logger"
	"github.com/AntonPashechko/ya-diplom/internal/models"
	"github.com/AntonPashechko/ya-diplom/internal/storage"
	"github.com/go-resty/resty/v2"
)

const (
	accrualSubURL    = "/api/orders/"
	registeredStatus = "REGISTERED"
)

type accrualChecker struct {
	tickerTime time.Duration
	endpoint   string
	client     *resty.Client
	storage    *storage.MartStorage
}

func NewAccrualChecker(cfg *config.Config, storage *storage.MartStorage) *accrualChecker {
	return &accrualChecker{
		tickerTime: time.Duration(cfg.AccrualInterval) * time.Second,
		endpoint:   cfg.AccrualAddress,
		client:     resty.New(),
		storage:    storage,
	}
}

func (m *accrualChecker) createURL(number string) string {
	return strings.Join([]string{m.endpoint, accrualSubURL, number}, "")
}

func (m *accrualChecker) Work(ctx context.Context) {

	ticker := time.NewTicker(m.tickerTime)

	for {
		select {
		// выход по ctx
		case <-ctx.Done():
			return
		//Обновляем статус заказов
		case <-ticker.C:
			m.checkOrders(ctx)
		}
	}
}

func (m *accrualChecker) checkOrders(ctx context.Context) {
	//Получим все заказы, у которых статус проверки не завершен
	numbers, err := m.storage.GetOrdersForCheck(ctx)
	if err != nil {
		logger.Error("cannot get orders for check: %s", err)
		return
	}

	//Для каждого заказа - проверим статус в accrual и обновим его в базе
	for _, number := range numbers {
		info, err := m.getAccrualInfo(number)
		if err != nil {
			logger.Error("cannot update accrual for number %s: %s", number, err)
			continue
		}

		if info.Status != registeredStatus {
			err := m.storage.UpdateOrderAccrual(ctx, number, info)
			if err != nil {
				logger.Error("cannot update accrual for number %s: %s", number, err)
			}
		}
	}
}

func (m *accrualChecker) getAccrualInfo(number string) (*models.AccrualDTO, error) {

	dto := &models.AccrualDTO{}

	resp, err := m.client.R().
		SetResult(dto).
		Get(m.createURL(number))

	if err != nil {
		return nil, fmt.Errorf("cannot get accrual for number %s: %w", number, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode())
	}

	return dto, nil
}
