package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestUser описывает структуру пользователя,
// которую ожидаем получить от /user/search.
type TestUser struct {
	ID         string `json:"id"`
	FirstName  string `json:"first_name"`
	SecondName string `json:"second_name"`
	Birthdate  string `json:"birthdate"`
	Biography  string `json:"biography"`
	City       string `json:"city"`
}

// LoginResponse описывает ответ /login.
type LoginResponse struct {
	Token string `json:"token"`
}

// doRequest — вспомогательная функция.
//
// Позволяет отправлять GET/POST-запросы
// без дублирования кода в каждом тесте.
func doRequest(
	t *testing.T,
	client *http.Client,
	method string,
	endpoint string,
	payload any,
) *http.Response {

	// Сообщаем Go, что это вспомогательная
	// функция для тестирования.
	t.Helper()

	var body *bytes.Reader

	if payload != nil {
		// Преобразуем Go-структуру в JSON.
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}

		body = bytes.NewReader(data)
	} else {
		body = bytes.NewReader(nil)
	}

	// Формируем HTTP-запрос.
	req, err := http.NewRequest(
		method,
		endpoint,
		body,
	)

	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	// Для POST с JSON указываем Content-Type.
	if payload != nil {
		req.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	// Отправляем запрос реальному серверу.
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	// Закрывать resp.Body должен вызывающий тест.
	return resp
}

// TestUserSearchAndLogin — основной интеграционный тест.
//
// Проверяет сценарий:
// 1. Найти тестового пользователя.
// 2. Получить его ID.
// 3. Авторизоваться.
// 4. Проверить отрицательные случаи.
func TestUserSearchAndLogin(t *testing.T) {

	// Интеграционные тесты запускаем явно.
	//
	// Обычный go test ./... не должен падать
	// из-за незапущенной PostgreSQL.
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("integration tests are disabled")
	}

	// Адрес нашего HTTP-сервера.
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	// Пароль передаётся через окружение,
	// а не хранится в исходном коде.
	password := os.Getenv("TEST_USER_PASSWORD")
	if password == "" {
		t.Fatal("TEST_USER_PASSWORD is not set")
	}

	// HTTP-клиент с общим таймаутом.
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// ID будет найден первым тестом.
	var userID int64

	// ==========================================
	// Тест №1. Поиск пользователя
	// ==========================================

	searchOK := t.Run(
		"search existing user",
		func(t *testing.T) {

			// Формируем query-параметры.
			params := url.Values{}
			params.Set("first_name", "Test")
			params.Set("last_name", "Test")

			endpoint := baseURL +
				"/user/search?" + params.Encode()

			resp := doRequest(
				t,
				client,
				http.MethodGet,
				endpoint,
				nil,
			)
			defer resp.Body.Close()

			// Проверяем HTTP-статус.
			if resp.StatusCode != http.StatusOK {
				t.Fatalf(
					"expected status 200, got %d",
					resp.StatusCode,
				)
			}

			// Декодируем JSON-массив.
			var users []TestUser

			if err := json.NewDecoder(resp.Body).
				Decode(&users); err != nil {

				t.Fatalf(
					"decode search response: %v",
					err,
				)
			}

			// Ищем именно нашу тестовую анкету
			// среди всех результатов поиска.
			var matchedUsers []TestUser

			for _, user := range users {
				if user.FirstName == "TestName" &&
					user.SecondName == "TestSecondName" {

					matchedUsers = append(matchedUsers, user)
				}
			}

			// Тестовая анкета должна существовать.
			if len(matchedUsers) == 0 {
				t.Fatal("test user was not found")
			}

			// Для надёжности требуем ровно одну
			// анкету с этим именем и фамилией.
			if len(matchedUsers) != 1 {
				t.Fatalf(
					"expected one test user, got %d",
					len(matchedUsers),
				)
			}

			user := matchedUsers[0]

			// Проверяем содержимое анкеты.
			if user.Birthdate != "2000-01-01" {
				t.Errorf(
					"unexpected birthdate: %q",
					user.Birthdate,
				)
			}

			if user.Biography != "Хобби, интересы и т.п." {
				t.Errorf(
					"unexpected biography: %q",
					user.Biography,
				)
			}

			if user.City != "Казань" {
				t.Errorf(
					"unexpected city: %q",
					user.City,
				)
			}

			// Преобразуем ID в int64.
			parsedID, err := strconv.ParseInt(
				user.ID, 10, 64,
			)
			if err != nil || parsedID <= 0 {
				t.Fatalf(
					"invalid user ID: %q",
					user.ID,
				)
			}

			userID = parsedID

			t.Logf("test user found, id=%d", userID)
		},
	)

	// Если поиск не прошёл, авторизоваться
	// по неизвестному ID мы не сможем.
	if !searchOK {
		return
	}

	// ==========================================
	// Тест №2. Успешная авторизация
	// ==========================================

	t.Run("login with valid password", func(t *testing.T) {

		payload := map[string]any{
			"id":       userID,
			"password": password,
		}

		resp := doRequest(
			t,
			client,
			http.MethodPost,
			baseURL+"/login",
			payload,
		)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf(
				"expected status 200, got %d",
				resp.StatusCode,
			)
		}

		var result LoginResponse

		if err := json.NewDecoder(resp.Body).
			Decode(&result); err != nil {

			t.Fatalf(
				"decode login response: %v",
				err,
			)
		}

		// Токен должен присутствовать.
		if result.Token == "" {
			t.Fatal("JWT token is empty")
		}

		// JWT в compact-формате содержит
		// три части, разделённые точками.
		if len(strings.Split(result.Token, ".")) != 3 {
			t.Fatal("invalid JWT token format")
		}

		// Сам токен не записываем в логи.
		t.Log("login successful, JWT received")
	})

	// ==========================================
	// Тест №3. Неправильный пароль
	// ==========================================

	t.Run("login with wrong password", func(t *testing.T) {

		payload := map[string]any{
			"id":       userID,
			"password": "definitely-wrong-password",
		}

		resp := doRequest(
			t,
			client,
			http.MethodPost,
			baseURL+"/login",
			payload,
		)
		defer resp.Body.Close()

		// Наш текущий /login возвращает 400
		// при несовпадении пароля.
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf(
				"expected status 400, got %d",
				resp.StatusCode,
			)
		}
	})

	// ==========================================
	// Тест №4. Нет обязательного last_name
	// ==========================================

	t.Run("search without last_name", func(t *testing.T) {

		params := url.Values{}
		params.Set("first_name", "Test")

		endpoint := baseURL +
			"/user/search?" + params.Encode()

		resp := doRequest(
			t,
			client,
			http.MethodGet,
			endpoint,
			nil,
		)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf(
				"expected status 400, got %d",
				resp.StatusCode,
			)
		}
	})

	// Сообщение после завершения сценария.
	t.Log(fmt.Sprintf(
		"API tests completed for user_id=%d",
		userID,
	))
}
