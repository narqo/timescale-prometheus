// This file and its contents are licensed under the Apache License 2.0.
// Please see the included NOTICE for copyright information and
// LICENSE for a copy of the license

package testhelpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultDB       = "postgres"
	connectTemplate = "postgres://postgres:password@%s:%d/%s"
)

var (
	PromHost          = "localhost"
	PromPort nat.Port = "9090/tcp"

	pgHost          = "localhost"
	pgPort nat.Port = "5432/tcp"
)

func pgConnectURL(dbName string) string {
	return fmt.Sprintf(connectTemplate, pgHost, pgPort.Int(), dbName)
}

// WithDB establishes a database for testing and calls the callback
func WithDB(t testing.TB, DBName string, f func(db *pgxpool.Pool, t testing.TB, connectString string)) {
	db, err := dbSetup(DBName)
	if err != nil {
		t.Fatal(err)
		return
	}
	defer func() {
		db.Close()
	}()
	f(db, t, pgConnectURL(DBName))
}

func GetReadOnlyConnection(t testing.TB, DBName string) *pgxpool.Pool {
	dbPool, err := pgxpool.Connect(context.Background(), pgConnectURL(DBName))
	if err != nil {
		t.Fatal(err)
	}

	_, err = dbPool.Exec(context.Background(), "SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY")
	if err != nil {
		t.Fatal(err)
	}

	return dbPool
}

func dbSetup(DBName string) (*pgxpool.Pool, error) {
	db, err := pgx.Connect(context.Background(), pgConnectURL(defaultDB))
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", DBName))
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", DBName))
	if err != nil {
		return nil, err
	}

	err = db.Close(context.Background())
	if err != nil {
		return nil, err
	}

	dbPool, err := pgxpool.Connect(context.Background(), pgConnectURL(DBName))
	if err != nil {
		return nil, err
	}
	return dbPool, nil
}

// StartPGContainer starts a postgreSQL container for use in testing
func StartPGContainer(ctx context.Context) (testcontainers.Container, error) {
	containerPort := nat.Port("5432/tcp")
	req := testcontainers.ContainerRequest{
		Image:        "timescale/timescaledb:latest-pg12",
		ExposedPorts: []string{string(containerPort)},
		WaitingFor:   wait.NewHostPortStrategy(containerPort),
		Env: map[string]string{
			"POSTGRES_PASSWORD": "password",
		},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	pgHost, err = container.Host(ctx)
	if err != nil {
		return nil, err
	}

	pgPort, err = container.MappedPort(ctx, containerPort)
	if err != nil {
		return nil, err
	}

	return container, nil
}

// StartPromContainer starts a Prometheus container for use in testing
func StartPromContainer(storagePath string, ctx context.Context) (testcontainers.Container, error) {
	// Set the storage directories permissions so Prometheus can write to them.
	err := os.Chmod(storagePath, 0777)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(filepath.Join(storagePath, "wal"), 0777); err != nil {
		return nil, err
	}

	prometheusPort := nat.Port("9090/tcp")

	req := testcontainers.ContainerRequest{
		Image:        "prom/prometheus",
		ExposedPorts: []string{string(prometheusPort)},
		WaitingFor:   wait.NewHostPortStrategy(prometheusPort),
		BindMounts: map[string]string{
			storagePath: "/prometheus",
		},
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	PromHost, err = container.Host(ctx)
	if err != nil {
		return nil, err
	}

	PromPort, err = container.MappedPort(ctx, prometheusPort)
	if err != nil {
		return nil, err
	}

	return container, nil
}
