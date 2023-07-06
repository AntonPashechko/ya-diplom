package handlers

import (
	"encoding/json"
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
	user_id, err := m.storage.Login(authDTO)
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
	currentUser := r.Context().Value("id").(string)

	//Нужно проверить, что заказа с таким номером не существует
	//А если есть - вернуть код, в зависимости от того, этого пользователя заказ или нет
	//200 — номер заказа уже был загружен этим пользователем;
	//409 — номер заказа уже был загружен другим пользователем;

	user_id, err := m.storage.GetExistOrderUser(string(numberData))
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

	err = m.storage.NewOrder(string(numberData), currentUser)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot create order exist: %s", err))
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (m *MartHandler) getOrders(w http.ResponseWriter, r *http.Request) {
	//Забираем id пользователя из контекста
	currentUser := r.Context().Value("id").(string)

	orders, err := m.storage.GetUserOrders(currentUser)
	if err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("cannot get user orders: %s", err))
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := json.NewEncoder(w).Encode(orders); err != nil {
		m.errorRespond(w, http.StatusInternalServerError, fmt.Errorf("error encoding response: %s", err))
	}
}

func (m *MartHandler) getBalance(w http.ResponseWriter, r *http.Request) {
}

func (m *MartHandler) addWithdraw(w http.ResponseWriter, r *http.Request) {
}

func (m *MartHandler) getWithdraws(w http.ResponseWriter, r *http.Request) {
}
