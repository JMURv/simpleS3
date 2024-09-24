package pg

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/JMURv/media-server/internal/cleaner"
	h "github.com/JMURv/media-server/internal/helpers"
	c "github.com/JMURv/media-server/pkg/config"
	"github.com/JMURv/media-server/pkg/consts"
	_ "github.com/lib/pq"
	"log"
	"path/filepath"
)

type PgCleaner struct {
	conf *c.Config
}

func New(conf *c.Config) cleaner.Cleaner {
	return &PgCleaner{
		conf: conf,
	}
}

func (c *PgCleaner) Clean(ctx context.Context) {
	db, err := sql.Open("postgres", fmt.Sprintf(
		"postgres://%s:%s@%s:%v/%s?sslmode=disable",
		c.conf.Postgres.User,
		c.conf.Postgres.Password,
		c.conf.Postgres.Host,
		c.conf.Postgres.Port,
		c.conf.Postgres.Database,
	))
	if err != nil {
		log.Fatalf("Error connecting to PostgreSQL: %s\n", err)
	}
	defer db.Close()

	pathsFromDB := make(map[string]struct{})
	for _, table := range c.conf.Postgres.Tables {
		err = getAllFilePathsFromDB(ctx, db, c.conf.Postgres.Field, table, pathsFromDB)
		if err != nil {
			log.Fatalf("Error retrieving file paths from DB: %s\n", err)
		}
	}

	localPaths, err := h.ListFilesInDir(consts.SavePath)
	if err != nil {
		log.Fatalf("Error listing files in directory: %s\n", err)
	}

	err = h.DeleteUnreferencedFiles(consts.SavePath, localPaths, pathsFromDB)
	if err != nil {
		log.Fatalf("Error deleting unreferenced files: %s\n", err)
	}

	log.Println("Files cleaned successfully")
}

func getAllFilePathsFromDB(ctx context.Context, db *sql.DB, field, table string, paths map[string]struct{}) error {
	query := fmt.Sprintf("SELECT %s FROM %s", field, table)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var filePath string
		if err := rows.Scan(&filePath); err != nil {
			return err
		}
		if filePath != "" && filepath.HasPrefix(filePath, "/uploads") {
			paths[filePath] = struct{}{}
		}
	}

	return rows.Err()
}
