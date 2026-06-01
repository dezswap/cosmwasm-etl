package db

import (
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"
)

type gormLogProbe struct {
	ID int
}

func TestOpenGormPostgresWithConn_DefaultLoggerIsSilent(t *testing.T) {
	output := triggerGormQueryError(t)

	require.Empty(t, strings.TrimSpace(output))
}

func TestOpenGormPostgresWithConn_ConfiguredLoggerLogsErrors(t *testing.T) {
	output := triggerGormQueryError(t, logger.Error)

	require.Contains(t, output, "boom")
	require.Contains(t, output, `SELECT * FROM "gorm_log_probes"`)
}

func TestGormLogLevelFromConfig(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		want      logger.LogLevel
		wantError bool
	}{
		{name: "empty defaults to silent", value: "", want: logger.Silent},
		{name: "silent", value: "silent", want: logger.Silent},
		{name: "error", value: "error", want: logger.Error},
		{name: "warn", value: "warn", want: logger.Warn},
		{name: "warning", value: "warning", want: logger.Warn},
		{name: "info", value: "info", want: logger.Info},
		{name: "case insensitive", value: " ERROR ", want: logger.Error},
		{name: "invalid", value: "debug", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GormLogLevelFromConfig(tt.value)
			if tt.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func triggerGormQueryError(t *testing.T, logLevels ...logger.LogLevel) string {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "gorm_log_probes"`)).
		WillReturnError(errors.New("boom"))

	output := captureStdout(t, func() {
		logLevel := logger.Silent
		if len(logLevels) > 0 {
			logLevel = logLevels[0]
		}

		gormDB, err := openGormPostgresWithConn(sqlDB, logLevel)
		require.NoError(t, err)

		err = gormDB.Find(&[]gormLogProbe{}).Error
		require.Error(t, err)
	})

	require.NoError(t, mock.ExpectationsWereMet())
	return output
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = writer
	defer func() {
		os.Stdout = oldStdout
	}()

	fn()

	require.NoError(t, writer.Close())
	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	return string(output)
}
