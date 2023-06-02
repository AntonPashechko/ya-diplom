package handlers

import (
	"fmt"
	"net/http"

	"github.com/AntonPashechko/ya-diplom/internal/logger"
	"github.com/AntonPashechko/ya-diplom/internal/models"
	"github.com/AntonPashechko/ya-diplom/internal/storage"
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
		r.Post("/register", m.userRegister)
		r.Post("/login", m.login)

		r.Route("/orders", func(r chi.Router) {
			//router.Use(jwt.Middleware)
			r.Post("/", m.addOrder)
			r.Get("/", m.getOrders)
		})

		r.Route("/balance", func(r chi.Router) {
			//router.Use(jwt.Middleware)
			r.Get("/", m.getBalance)
			r.Post("/withdraw", m.addWithdraw)
		})

		r.Route("/withdrawals", func(r chi.Router) {
			//router.Use(jwt.Middleware)
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
	if m.storage.IsUserExist(authDTO.Login) {
		m.errorRespond(w, http.StatusConflict, fmt.Errorf("user with login %s already exist", authDTO.Login))
		return
	}
	//Создаем пользователя, получаем идентификатор для токена
	user_id, err := m.storage.CreateUser(authDTO)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("user with login %s already exist", authDTO.Login))
		return
	}

	fmt.Println(user_id)
}

func (m *MartHandler) login(w http.ResponseWriter, r *http.Request) {
	//Разобрали запрос
	authDTO, err := models.NewDTO[models.AuthDTO](r.Body)
	if err != nil {
		m.errorRespond(w, http.StatusBadRequest, fmt.Errorf("cannot decode auth dto: %s", err))
		return
	}

	//Создаем пользователя, получаем идентификатор для токена
	user_id, err := m.storage.Login(authDTO)
	if err != nil {
		m.errorRespond(w, http.StatusUnauthorized, fmt.Errorf("authentication failed: %w", err))
		return
	}

	fmt.Println(user_id)
}

func (m *MartHandler) addOrder(w http.ResponseWriter, r *http.Request) {
}

func (m *MartHandler) getOrders(w http.ResponseWriter, r *http.Request) {
}

func (m *MartHandler) getBalance(w http.ResponseWriter, r *http.Request) {
}

func (m *MartHandler) addWithdraw(w http.ResponseWriter, r *http.Request) {
}

func (m *MartHandler) getWithdraws(w http.ResponseWriter, r *http.Request) {
}
