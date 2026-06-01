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

func TestOpenGormPostgresWithConn_LogLevelOptionOverridesDefault(t *testing.T) {
	output := triggerGormQueryError(t, func() GormOption {
		return WithGormLogLevel(logger.Error)
	})

	require.Contains(t, output, "boom")
	require.Contains(t, output, `SELECT * FROM "gorm_log_probes"`)
}

func triggerGormQueryError(t *testing.T, optFactories ...func() GormOption) string {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "gorm_log_probes"`)).
		WillReturnError(errors.New("boom"))

	output := captureStdout(t, func() {
		opts := make([]GormOption, 0, len(optFactories))
		for _, factory := range optFactories {
			opts = append(opts, factory())
		}

		gormDB, err := OpenGormPostgresWithConn(sqlDB, opts...)
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
