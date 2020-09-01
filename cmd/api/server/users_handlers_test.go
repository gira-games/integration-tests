package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/asankov/gira/internal/auth"
	"github.com/asankov/gira/internal/fixtures"
	"github.com/asankov/gira/pkg/models"
	"github.com/asankov/gira/pkg/models/postgres"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

var (
	expectedUser = models.User{
		Username: "test",
		Email:    "test@test.com",
		Password: "t3$T123",
	}
)

func newServer(t *testing.T, opts *Options) *Server {
	if opts == nil {
		opts = &Options{}
	}
	opts.Log = logrus.StandardLogger()

	srv, err := New(opts)
	if err != nil {
		t.Fatalf("Got unexpected error while constructing server: %v", err)
	}
	return srv
}

func TestUserCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userModel := fixtures.NewUserModelMock(ctrl)
	authenticator := fixtures.NewAuthenticatorMock(ctrl)
	srv := newServer(t, &Options{
		UserModel:     userModel,
		Authenticator: authenticator,
	})

	userModel.EXPECT().
		Insert(&expectedUser).
		Return(&expectedUser, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/users", fixtures.Marshall(t, expectedUser))
	srv.ServeHTTP(w, r)

	got, expected := w.Code, http.StatusOK
	if got != expected {
		t.Fatalf("Got (%d) for status code, expected (%d)", got, expected)
	}

	var user models.User
	fixtures.Decode(t, w.Body, &user)
	if user.Username != expectedUser.Username {
		t.Errorf("Got (%s) for username, expected (%s)", user.Username, expectedUser.Username)
	}
	if user.Email != expectedUser.Email {
		t.Errorf("Got (%s) for email, expected (%s)", user.Email, expectedUser.Email)
	}
}

func TestUserCreateValidationError(t *testing.T) {
	cases := []struct {
		name string
		user *models.User
	}{
		{
			name: "No username",
			user: &models.User{
				Email:    "test@test.com",
				Password: "t3$t",
			},
		},
		{
			name: "No email",
			user: &models.User{
				Username: "test",
				Password: "t3$t",
			},
		},
		{
			name: "No password",
			user: &models.User{
				Username: "test",
				Email:    "test@test.com",
			},
		},
		{
			name: "Filled ID",
			user: &models.User{
				ID:       "1",
				Username: "test",
				Email:    "test@test.com",
				Password: "t3$t",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := newServer(t, nil)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users", fixtures.Marshall(t, c.user))
			srv.ServeHTTP(w, r)

			got, expected := w.Code, http.StatusBadRequest
			if got != expected {
				t.Fatalf("Got (%d) for status code, expected (%d)", got, expected)
			}
		})
	}
}

func TestUserCreateEmptyBody(t *testing.T) {
	srv := newServer(t, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/users", nil)
	srv.ServeHTTP(w, r)

	got, expected := w.Code, http.StatusBadRequest
	if got != expected {
		t.Fatalf("Got (%d) for status code, expected (%d)", got, expected)
	}
}

func TestUserCreateDBError(t *testing.T) {
	cases := []struct {
		name         string
		dbError      error
		expectedCode int
	}{
		{
			name:         "Email already exists",
			dbError:      postgres.ErrEmailAlreadyExists,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Name already exists",
			dbError:      postgres.ErrUsernameAlreadyExists,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Unknown error",
			dbError:      errors.New("unknown error"),
			expectedCode: http.StatusInternalServerError,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userModel := fixtures.NewUserModelMock(ctrl)

			srv := newServer(t, &Options{
				UserModel: userModel,
			})

			userModel.EXPECT().
				Insert(&expectedUser).
				Return(nil, c.dbError)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users", fixtures.Marshall(t, expectedUser))
			srv.ServeHTTP(w, r)

			got, expected := w.Code, c.expectedCode
			if got != expected {
				t.Fatalf("Got (%d) for status code, expected (%d)", got, expected)
			}
		})
	}
}

func TestUserLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userModel := fixtures.NewUserModelMock(ctrl)
	authenticatorMock := fixtures.NewAuthenticatorMock(ctrl)

	srv := newServer(t, &Options{
		UserModel:     userModel,
		Authenticator: authenticatorMock,
	})

	userModel.EXPECT().
		Authenticate(expectedUser.Email, expectedUser.Password).
		Return(&expectedUser, nil)
	userModel.EXPECT().
		AssociateTokenWithUser(expectedUser.ID, token).
		Return(nil)

	token := "my_test_token"
	authenticatorMock.EXPECT().
		NewTokenForUser(&expectedUser).
		Return(token, nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/users/login", fixtures.Marshall(t, expectedUser))
	srv.ServeHTTP(w, r)

	got, expected := w.Code, http.StatusOK
	if got != expected {
		t.Fatalf("Got (%d) for status code, expected (%d)", got, expected)
	}
	var userResponse models.UserLoginResponse
	fixtures.Decode(t, w.Body, &userResponse)
	if userResponse.Token != token {
		t.Fatalf(`Got ("%s") for token, expected ("%s")`, userResponse.Token, token)
	}
}

func TestUserLoginValidationError(t *testing.T) {
	testCases := []struct {
		name string
		user *models.User
	}{
		{
			name: "No email",
			user: &models.User{
				Email:    "",
				Password: "T3$T",
			},
		},
		{
			name: "No password",
			user: &models.User{
				Email:    "test@mail.com",
				Password: "",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userModel := fixtures.NewUserModelMock(ctrl)

			srv := newServer(t, &Options{
				UserModel: userModel,
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users/login", fixtures.Marshall(t, testCase.user))
			srv.ServeHTTP(w, r)

			got, expected := w.Code, http.StatusBadRequest
			if got != expected {
				t.Fatalf("Got (%d) for status code, expected (%d)", got, expected)
			}
			// TODO: assert body, once we start returning proper errors
		})
	}
}

func TestUserLoginServiceError(t *testing.T) {
	testCases := []struct {
		name         string
		setup        func(u *fixtures.UserModelMock, a *fixtures.AuthenticatorMock)
		expectedCode int
	}{
		{
			name: "UserModel.Authenticate fails",
			setup: func(u *fixtures.UserModelMock, a *fixtures.AuthenticatorMock) {
				u.EXPECT().
					Authenticate(expectedUser.Email, expectedUser.Password).
					Return(nil, errors.New("user not found"))
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Authenticator.NewTokenForUser fails",
			setup: func(u *fixtures.UserModelMock, a *fixtures.AuthenticatorMock) {
				u.EXPECT().
					Authenticate(expectedUser.Email, expectedUser.Password).
					Return(&expectedUser, nil)

				a.EXPECT().
					NewTokenForUser(&expectedUser).
					Return("", errors.New("intentional error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userModel := fixtures.NewUserModelMock(ctrl)
			authenticatorMock := fixtures.NewAuthenticatorMock(ctrl)

			testCase.setup(userModel, authenticatorMock)

			srv := newServer(t, &Options{
				UserModel:     userModel,
				Authenticator: authenticatorMock,
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/users/login", fixtures.Marshall(t, expectedUser))
			srv.ServeHTTP(w, r)

			got, expected := w.Code, testCase.expectedCode
			if got != expected {
				t.Fatalf("Got (%d) for status code, expected (%d)", got, expected)
			}
		})
	}
}

func TestUserGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userModelMock := fixtures.NewUserModelMock(ctrl)
	authenticatorMock := fixtures.NewAuthenticatorMock(ctrl)

	authenticatorMock.EXPECT().
		DecodeToken(gomock.Eq(token)).
		Return(nil, nil)
	userModelMock.EXPECT().
		GetUserByToken(gomock.Eq(token)).
		Return(&expectedUser, nil)

	srv := newServer(t, &Options{
		UserModel:     userModelMock,
		Authenticator: authenticatorMock,
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users", nil)
	r.Header.Add("x-auth-token", token)
	srv.ServeHTTP(w, r)

	got, expected := w.Code, http.StatusOK
	if got != expected {
		t.Fatalf("Got (%d) for status code, expected (%d)", got, expected)
	}
	var userResponse models.UserResponse
	fixtures.Decode(t, w.Body, &userResponse)
	gotUser := userResponse.User

	if gotUser.ID != expectedUser.ID {
		t.Errorf("Got %s user ID, expected %s", gotUser.ID, expectedUser.ID)
	}
	if gotUser.Username != expectedUser.Username {
		t.Errorf("Got %s username, expected %s", gotUser.Username, expectedUser.Username)
	}
	if gotUser.Email != expectedUser.Email {
		t.Errorf("Got %s email, expected %s", gotUser.Email, expectedUser.Email)
	}
}

func TestUserGetUnathorized(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(*fixtures.AuthenticatorMock, *fixtures.UserModelMock, *http.Request)
	}{
		{
			name:  "No token",
			setup: func(a *fixtures.AuthenticatorMock, u *fixtures.UserModelMock, r *http.Request) {},
		},
		{
			name: "Invalid signature",
			setup: func(a *fixtures.AuthenticatorMock, u *fixtures.UserModelMock, r *http.Request) {
				r.Header.Add("x-auth-token", token)

				a.EXPECT().
					DecodeToken(gomock.Eq(token)).
					Return(nil, auth.ErrInvalidSignature)
			},
		},
		{
			name: "Token expired",
			setup: func(a *fixtures.AuthenticatorMock, u *fixtures.UserModelMock, r *http.Request) {
				r.Header.Add("x-auth-token", token)

				a.EXPECT().
					DecodeToken(gomock.Eq(token)).
					Return(nil, auth.ErrTokenExpired)
			},
		},
		{
			name: "Generic token error",
			setup: func(a *fixtures.AuthenticatorMock, u *fixtures.UserModelMock, r *http.Request) {
				r.Header.Add("x-auth-token", token)

				a.EXPECT().
					DecodeToken(gomock.Eq(token)).
					Return(nil, errors.New("some generic error"))
			},
		},
		{
			name: "UserModel error",
			setup: func(a *fixtures.AuthenticatorMock, u *fixtures.UserModelMock, r *http.Request) {
				r.Header.Add("x-auth-token", token)

				a.EXPECT().
					DecodeToken(gomock.Eq(token)).
					Return(nil, nil)
				u.EXPECT().
					GetUserByToken(gomock.Eq(token)).
					Return(nil, errors.New("some error"))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			authenticatorMock := fixtures.NewAuthenticatorMock(ctrl)
			userModelMock := fixtures.NewUserModelMock(ctrl)
			srv := newServer(t, &Options{
				Authenticator: authenticatorMock,
				UserModel:     userModelMock,
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/users", nil)
			testCase.setup(authenticatorMock, userModelMock, r)
			srv.ServeHTTP(w, r)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Got %d for status, expected %d", w.Code, http.StatusUnauthorized)
			}
			// TODO: assert error once we return JSON
		})
	}
}
