package passwordreset

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type capturedDelivery struct { recipient, resetURL string; calls int }
func (d *capturedDelivery) DeliverPasswordReset(_ context.Context, recipient, resetURL string) error { d.recipient, d.resetURL, d.calls = recipient, resetURL, d.calls+1; return nil }

func TestRequestIsUniformForUnknownEmail(t *testing.T) {
	h, mock, delivery := testHandler(t)
	mock.ExpectQuery("SELECT id,email::text email FROM users").WithArgs("missing@example.test").WillReturnRows(sqlmock.NewRows([]string{"id", "email"}))
	r := request(t, h, http.MethodPost, "/api/v1/auth/password-resets", `{"Email":"missing@example.test"}`)
	require.Equal(t, http.StatusAccepted, r.Code)
	require.JSONEq(t, `{"data":{"accepted":true}}`, r.Body.String())
	require.Zero(t, delivery.calls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestStoresOnlyDigestAndDeliversRawToken(t *testing.T) {
	h, mock, delivery := testHandler(t)
	uid := uuid.New()
	mock.ExpectQuery("SELECT id,email::text email FROM users").WithArgs("user@example.test").WillReturnRows(sqlmock.NewRows([]string{"id", "email"}).AddRow(uid, "user@example.test"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE password_reset_tokens SET used_at").WithArgs(sqlmock.AnyArg(), uid).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO password_reset_tokens").WithArgs(uid, sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	r := request(t, h, http.MethodPost, "/api/v1/auth/password-resets", `{"Email":"user@example.test"}`)
	require.Equal(t, http.StatusAccepted, r.Code)
	require.Equal(t, 1, delivery.calls)
	require.Equal(t, "user@example.test", delivery.recipient)
	require.Contains(t, delivery.resetURL, "http://app.example.test/reset-password?token=")
	require.NotContains(t, delivery.resetURL, uid.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConsumeRejectsExpiredTokenWithoutMutation(t *testing.T) {
	h, mock, _ := testHandler(t)
	h.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC) }
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id,user_id,expires_at FROM password_reset_tokens").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at"}).AddRow(uuid.New(), uuid.New(), h.now().Add(-time.Minute)))
	mock.ExpectRollback()
	r := request(t, h, http.MethodPost, "/api/v1/auth/password-resets/consume", `{"Token":"expired","Password":"a-secure-password-2026"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConsumeChangesPasswordRevokesSessionsAndCannotRepeat(t *testing.T) {
	h, mock, _ := testHandler(t)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC); h.now = func() time.Time { return now }
	tokenID, userID := uuid.New(), uuid.New()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id,user_id,expires_at FROM password_reset_tokens").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at"}).AddRow(tokenID, userID, now.Add(time.Hour)))
	mock.ExpectExec("UPDATE user_credentials").WithArgs(sqlmock.AnyArg(), now, now, userID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE user_sessions SET revoked_at").WithArgs(now, userID).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE password_reset_tokens SET used_at").WithArgs(now, tokenID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	r := request(t, h, http.MethodPost, "/api/v1/auth/password-resets/consume", `{"Token":"valid-once","Password":"a-secure-password-2026"}`)
	require.Equal(t, http.StatusOK, r.Code)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id,user_id,expires_at FROM password_reset_tokens").WithArgs(sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at"}))
	mock.ExpectRollback()
	r = request(t, h, http.MethodPost, "/api/v1/auth/password-resets/consume", `{"Token":"valid-once","Password":"another-password-2026"}`)
	require.Equal(t, http.StatusUnprocessableEntity, r.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func testHandler(t *testing.T) (*Handler, sqlmock.Sqlmock, *capturedDelivery) {
	t.Helper(); gin.SetMode(gin.TestMode)
	sqlDB, mock, err := sqlmock.New(); require.NoError(t, err); t.Cleanup(func(){ _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing:true, Logger:logger.Default.LogMode(logger.Silent)}); require.NoError(t, err)
	delivery := &capturedDelivery{}
	h, err := New(db, delivery, "http://app.example.test"); require.NoError(t, err)
	return h, mock, delivery
}

func request(t *testing.T, h *Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper(); router := gin.New(); RegisterRoutes(router, h)
	w := httptest.NewRecorder(); req := httptest.NewRequest(method, path, strings.NewReader(body)); req.Header.Set("Content-Type", "application/json"); router.ServeHTTP(w, req)
	if w.Body.Len() > 0 { var value any; require.NoError(t, json.Unmarshal(w.Body.Bytes(), &value)) }
	return w
}
