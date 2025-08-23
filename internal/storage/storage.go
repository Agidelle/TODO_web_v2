package storage

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/agidelle/merch-shop/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	pool *pgxpool.Pool
}

func NewPool(ctx context.Context, dsn string) *Storage {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse database URL: %v\n", err)
	}

	cfg.MaxConns = 10 // Set a maximum number of connections
	cfg.MinConns = 1  // Set a minimum number of connections
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}

	// Check if the connection pool is working
	err = pool.Ping(ctx)
	if err != nil {
		pool.Close()
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	return &Storage{pool: pool}
}

func (s *Storage) CloseDB() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Storage) FindTask(ctx context.Context, filter *domain.Filter) ([]*domain.Task, error) {
	tasks := make([]*domain.Task, 0)
	query := "SELECT id, date, title, comment, repeat FROM scheduler"
	args := []interface{}{}
	conditions := []string{}
	argIdx := 1

	//Добавление условий в зависимости от фильтра
	if filter.ID != nil {
		conditions = append(conditions, "id = $"+strconv.Itoa(argIdx))
		args = append(args, *filter.ID)
		argIdx++
	}
	if filter.SearchTerm != "" {
		searchPattern := "%" + filter.SearchTerm + "%"
		conditions = append(conditions, "(title ILIKE $"+strconv.Itoa(argIdx)+" OR comment ILIKE $"+strconv.Itoa(argIdx+1)+")")
		args = append(args, searchPattern, searchPattern)
		argIdx += 2
	}
	if filter.Date != "" {
		conditions = append(conditions, "date = $"+strconv.Itoa(argIdx))
		args = append(args, filter.Date)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY date"
	if filter.Limit > 0 {
		query += " LIMIT $" + strconv.Itoa(argIdx)
		args = append(args, filter.Limit)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var t domain.Task
		err = rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *Storage) CreateTask(ctx context.Context, task *domain.Task) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, "INSERT INTO scheduler (date, title, comment, repeat) VALUES ($1,$2,$3,$4) RETURNING id",
		task.Date, task.Title, task.Comment, task.Repeat).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Storage) UpdateTask(ctx context.Context, task *domain.Task) error {
	res, err := s.pool.Exec(ctx,
		"UPDATE scheduler SET date = $1, title = $2, comment = $3, repeat = $4 WHERE id = $5",
		task.Date, task.Title, task.Comment, task.Repeat, task.ID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("id задачи не найден в БД")
	}
	return nil
}

func (s *Storage) DeleteTask(ctx context.Context, id *int) error {
	res, err := s.pool.Exec(ctx,
		"DELETE FROM scheduler WHERE id = $1", id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("id задачи не найден в БД")
	}
	return nil
}
