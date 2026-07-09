package postgres

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	// Драйверы регистрируются через init: pgx/v5 под схемой "pgx5", источник — файлы.
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations применяет все up-миграции из каталога migrationsPath.
// Отсутствие новых миграций ошибкой не считается.
func RunMigrations(dsn, migrationsPath string) (err error) {
	// golang-migrate выбирает драйвер по схеме DSN: для pgx/v5 это "pgx5".
	migrateDSN := strings.Replace(dsn, "postgres://", "pgx5://", 1)

	m, err := migrate.New("file://"+migrationsPath, migrateDSN)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		err = errors.Join(err, srcErr, dbErr)
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
