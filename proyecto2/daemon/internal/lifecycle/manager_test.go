package lifecycle

import (
	"errors"
	"os/user"
	"reflect"
	"testing"
)

func TestResolveCronUser(t *testing.T) {
	lookupFailure := errors.New("usuario inexistente")

	tests := []struct {
		name         string
		effectiveUID int
		sudoUser     string
		account      *user.User
		lookupError  error
		expected     string
		expectError  bool
		expectLookup bool
	}{
		{
			name:         "usuario normal ignora entorno sudo",
			effectiveUID: 1000,
			sudoUser:     "falso",
		},
		{
			name:         "root directo conserva crontab root",
			effectiveUID: 0,
		},
		{
			name:         "sudo root conserva crontab root",
			effectiveUID: 0,
			sudoUser:     "root",
		},
		{
			name:         "sudo usa usuario validado",
			effectiveUID: 0,
			sudoUser:     "victor",
			account: &user.User{
				Uid:      "1000",
				Username: "victor",
			},
			expected:     "victor",
			expectLookup: true,
		},
		{
			name:         "alias con UID cero conserva root",
			effectiveUID: 0,
			sudoUser:     "administrador",
			account: &user.User{
				Uid:      "0",
				Username: "administrador",
			},
			expectLookup: true,
		},
		{
			name:         "usuario sudo inexistente",
			effectiveUID: 0,
			sudoUser:     "inexistente",
			lookupError:  lookupFailure,
			expectError:  true,
			expectLookup: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lookupCalled := false
			lookup := func(name string) (*user.User, error) {
				lookupCalled = true
				if name != test.sudoUser {
					t.Fatalf(
						"lookup recibió %q; se esperaba %q",
						name,
						test.sudoUser,
					)
				}

				return test.account, test.lookupError
			}

			actual, err := resolveCronUser(
				test.effectiveUID,
				test.sudoUser,
				lookup,
			)
			if (err != nil) != test.expectError {
				t.Fatalf(
					"resolveCronUser() error = %v; esperado error=%v",
					err,
					test.expectError,
				)
			}
			if actual != test.expected {
				t.Fatalf(
					"resolveCronUser() = %q; esperado %q",
					actual,
					test.expected,
				)
			}
			if lookupCalled != test.expectLookup {
				t.Fatalf(
					"lookup llamado=%v; esperado %v",
					lookupCalled,
					test.expectLookup,
				)
			}
		})
	}
}

func TestCronCommand(t *testing.T) {
	tests := []struct {
		name         string
		cronUser     string
		expectedName string
		expectedArgs []string
	}{
		{
			name:         "usuario actual",
			expectedName: "bash",
			expectedArgs: []string{"/proyecto/cron/install_cron.sh"},
		},
		{
			name:         "usuario original de sudo",
			cronUser:     "victor",
			expectedName: "sudo",
			expectedArgs: []string{
				"-u",
				"victor",
				"--",
				"bash",
				"/proyecto/cron/install_cron.sh",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name, arguments := cronCommand(
				test.cronUser,
				"/proyecto/cron/install_cron.sh",
			)
			if name != test.expectedName {
				t.Fatalf(
					"nombre = %q; esperado %q",
					name,
					test.expectedName,
				)
			}
			if !reflect.DeepEqual(arguments, test.expectedArgs) {
				t.Fatalf(
					"argumentos = %#v; esperados %#v",
					arguments,
					test.expectedArgs,
				)
			}
		})
	}
}
