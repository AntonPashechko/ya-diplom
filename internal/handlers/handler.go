package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/AntonPashechko/ya-diplom/internal/auth"
	"github.com/AntonPashechko/ya-diplom/internal/logger"
	"github.com/AntonPashechko/ya-diplom/internal/models"
	"github.com/AntonPashechko/ya-diplom/internal/storage"
	"github.com/AntonPashechko/ya-diplom/pkg/utils"
	"github.com/go-chi/chi/v5"
)

type MartHandler struct {
	storage *storage.MartStorage
}

func NewMartHandler(storage *storage.MartStorage) MartHandler {
	return MartHandler{
		storage: storage,
	}
}

func (m *MartHandler) Register(r *chi.Mux) {

	r.Route("/api/user", func(r chi.Router) {
		//Регистрация нового пользователя
		r.Post("/register", m.userRegister)
		//Аутентификация существующего пользователя
		r.Post("/login", m.login)

		r.Route("/orders", func(r chi.Router) {
			r.Use(auth.Middleware)
			r.Post("/", m.addOrder)
			r.Get("/", m.getOrders)
		})

		r.Route("/balance", func(r chi.Router) {
			r.Use(auth.Middleware)
			r.Get("/", m.getBalance)
			r.Post("/withdraw", m.addWithdraw)
		})

		r.Route("/withdrawals", func(r chi.Router) {
			r.Use(auth.Middleware)
			r.Get("/", m.getWithdraws)
		})
	})
}

func (m *MartHandler) errorRespond(w http.ResponseWriter, code int, err error) {
	logger.Error(err.Error())
	w.WriteHeader(code)
}

func (m *MartHandler) userRegister(w http.ResponseWriter, r *http.Request) {

	//Разобрали запрос
	authDTO, err := models.NewDTO[models.AuthDTO](r.Body)
	if err != nil {
		m.errorRespond(w, http.StatusBadRequest, fmt.Errorf("cannot decode auth dto: %s", err))
		return
	}
	//Проверили наличие полей
	if err := authDTO.Validate(); err != nil {
		m.errorRespond(w, http.StatusBadRequest, fmt.Errorf("cannot validate auth dto: %s", err))
		return
	}
	//Проверяем, что пользака с таким логином нет
	if m.storage.IsUserExist(r.Context(), authDTO.Login) {
		m.errorRespond(w, http.StatusConflict, fmt.Errorf("user with login %s already exist", authDTO.Login))
		return
	}
	//Создаем пользователя, получаем идентификатор для токена
	user_id, err := m.storage.CreateUser(r.Context(), authDTO)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot create new user: %s", err))
		return
	}

	//Выпускаем токен, посылаем в заголовке ответа
	jwt, err := auth.CreateToken(user_id)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot create jwt: %s", err))
		return
	}

	w.Header().Set("Authorization", jwt)
}

func (m *MartHandler) login(w http.ResponseWriter, r *http.Request) {
	//Разобрали запрос
	authDTO, err := models.NewDTO[models.AuthDTO](r.Body)
	if err != nil {
		m.errorRespond(w, http.StatusBadRequest, fmt.Errorf("cannot decode auth dto: %s", err))
		return
	}

	//Провереяем корректность данных пользователя
	user_id, err := m.storage.Login(r.Context(), authDTO)
	if err != nil {
		m.errorRespond(w, http.StatusUnauthorized, fmt.Errorf("authentication failed: %s", err))
		return
	}

	//Выпускаем токен, посылаем в заголовке ответа
	jwt, err := auth.CreateToken(user_id)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot create jwt: %s", err))
		return
	}

	w.Header().Set("Authorization", jwt)
}

func (m *MartHandler) addOrder(w http.ResponseWriter, r *http.Request) {

	//Прочитаем тело запроса
	numberData, err := io.ReadAll(r.Body)
	if err != nil {
		m.errorRespond(w, http.StatusBadRequest, fmt.Errorf("cannot get request body: %s", err))
		return
	}

	//Проверим, что там число и не пустая строка
	orderNumber, err := utils.StrToInt64(string(numberData))
	if err != nil || orderNumber == 0 {
		m.errorRespond(w, http.StatusUnprocessableEntity, fmt.Errorf("bad order number: %s", err))
		return
	}

	//Проверим по алгоритму Луна
	if !utils.ValidLuhn(orderNumber) {
		m.errorRespond(w, http.StatusUnprocessableEntity, fmt.Errorf("bad Luhn check"))
		return
	}

	//Забираем id пользователя из контекста
	currentUser := r.Context().Value("user").(string)

	//Нужно проверить, что заказа с таким номером не существует
	//А если есть - вернуть код, в зависимости от того, этого пользователя заказ или нет
	//200 — номер заказа уже был загружен этим пользователем;
	//409 — номер заказа уже был загружен другим пользователем;

	user_id, err := m.storage.GetExistOrderUser(r.Context(), string(numberData))
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot check number exist: %s", err))
		return
	}
	if user_id != `` {
		if user_id == currentUser {
			return
		} else {
			m.errorRespond(w, http.StatusConflict, fmt.Errorf("other user have this order number"))
			return
		}
	}

	err = m.storage.NewOrder(r.Context(), string(numberData), currentUser)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot create new order: %s", err))
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (m *MartHandler) getOrders(w http.ResponseWriter, r *http.Request) {
	//Забираем id пользователя из контекста
	currentUser := r.Context().Value("user").(string)

	orders, err := m.storage.GetUserOrders(r.Context(), currentUser)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot get user orders: %s", err))
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(orders); err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("error encoding response: %s", err))
	}
}

func (m *MartHandler) getBalance(w http.ResponseWriter, r *http.Request) {

	//Забираем id пользователя из контекста
	currentUser := r.Context().Value("user").(string)

	//Получаем текущий баланс пользователя
	balance, err := m.storage.GetUserBalance(r.Context(), currentUser)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot get user balance: %s", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(balance); err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("error encoding response: %s", err))
	}
}

func (m *MartHandler) addWithdraw(w http.ResponseWriter, r *http.Request) {

	//Разобрали запрос
	dto, err := models.NewDTO[models.WithdrawDTO](r.Body)
	if err != nil {
		m.errorRespond(w, http.StatusBadRequest, fmt.Errorf("cannot decode withdraw dto: %s", err))
		return
	}

	//Проверим, что там число и не пустая строка
	orderNumber, err := utils.StrToInt64(string(dto.Order))
	if err != nil || orderNumber == 0 {
		m.errorRespond(w, http.StatusUnprocessableEntity, fmt.Errorf("bad order number: %s", err))
		return
	}

	//Проверим по алгоритму Луна
	if !utils.ValidLuhn(orderNumber) {
		m.errorRespond(w, http.StatusUnprocessableEntity, fmt.Errorf("bad Luhn check"))
		return
	}

	//Забираем id пользователя из контекста
	currentUser := r.Context().Value("user").(string)

	err = m.storage.AddWithdraw(r.Context(), dto, currentUser)
	if err != nil {
		if errors.Is(err, storage.ErrNotEnoughFunds) {
			m.errorRespond(w, http.StatusPaymentRequired, err)
			return
		}
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot create new withdraw: %s", err))
		return
	}
}

func (m *MartHandler) getWithdraws(w http.ResponseWriter, r *http.Request) {
	//Забираем id пользователя из контекста
	currentUser := r.Context().Value("user").(string)

	withdraws, err := m.storage.GetUserWithdraws(r.Context(), currentUser)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot get user withdraws: %s", err))
		return
	}

	if len(withdraws) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(withdraws); err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("error encoding response: %s", err))
	}
}
