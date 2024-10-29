package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

// 发送验证码与登录流程测试

func TestLoginOrSignupRoutine(t *testing.T) {
	conditionallyWaitForFormerTest()
	conditionallyStartServer(t)
	defer shutdownServer(t)

	userDAO := dao.NewUserDAO(config.DB)
	userCache := cache.NewUserCache(config.RedisClient)

	testCases := []struct {
		name           string
		phone          string
		setupFunc      func(context.Context) error
		cleanupFunc    func(context.Context) error
		expectedStatus int
		checkResponse  func(*testing.T, types.LoginOrSignupResp)
	}{
		{
			name:  "New user signup",
			phone: "+8611111111111",
			setupFunc: func(ctx context.Context) error {
				return nil // No setup needed for new user
			},
			cleanupFunc: func(ctx context.Context) error {
				err := userDAO.DeleteUserByPhone(ctx, "+8611111111111")
				if err != nil {
					return err
				}
				return userCache.DeleteAuthCodeAndLimit(ctx, "+8611111111111")
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, result types.LoginOrSignupResp) {
				assert.NotEmpty(t, result.UUID)
				assert.NotEmpty(t, result.Token)
			},
		},
		{
			name:  "Existing user login",
			phone: "+8611111111111",
			setupFunc: func(ctx context.Context) error {
				user := &model.User{
					Phone:     "+8611111111111",
					UUID:      "existing-user-uuid",
					MatchList: "[]",
					BlackList: "[]",
				}
				return userDAO.CreateUser(ctx, user)
			},
			cleanupFunc: func(ctx context.Context) error {
				err := userDAO.DeleteUserByPhone(ctx, "+8611111111111")
				if err != nil {
					return err
				}
				return userCache.DeleteAuthCodeAndLimit(ctx, "+8611111111111")
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, result types.LoginOrSignupResp) {
				assert.NotEmpty(t, result.UUID)
				assert.NotEmpty(t, result.Token)
			},
		},
	}

	client := &http.Client{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			defer tc.cleanupFunc(ctx)

			require.NoError(t, tc.setupFunc(ctx))

			sendAuthCodeReq := types.SendAuthCodeReq{
				Phone: tc.phone,
			}
			sendAuthCodeBody, err := json.Marshal(sendAuthCodeReq)
			require.NoError(t, err)

			// 发送业务接口的HTTP请求可以封装成函数
			sendAuthCodeURL := fmt.Sprintf("http://%s%s%s", testConfig.ServerAddr, controller.UrlV1, controller.UrlSendAuthCode)
			sendAuthCodeResp, err := client.Post(sendAuthCodeURL, "application/json", bytes.NewBuffer(sendAuthCodeBody))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, sendAuthCodeResp.StatusCode)
			sendAuthCodeResp.Body.Close()

			authCode, err := userCache.GetAuthCode(ctx, tc.phone)
			require.NoError(t, err)

			loginReq := types.LoginOrSignupReq{
				Phone: tc.phone,
				Code:  authCode,
			}
			loginBody, err := json.Marshal(loginReq)
			require.NoError(t, err)

			loginURL := fmt.Sprintf("http://%s%s%s", testConfig.ServerAddr, controller.UrlV1, controller.UrlLoginOrSignup)
			loginResp, err := client.Post(loginURL, "application/json", bytes.NewBuffer(loginBody))
			require.NoError(t, err)
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					fmt.Println("Failed to close response body", err)
				}
			}(loginResp.Body)

			assert.Equal(t, tc.expectedStatus, loginResp.StatusCode)

			var result types.LoginOrSignupResp
			err = json.NewDecoder(loginResp.Body).Decode(&result)
			require.NoError(t, err)

			tc.checkResponse(t, result)
		})
	}
}
