package cleaner

import (
	"context"
	"github.com/JMURv/media-server/internal/cleaner"
	h "github.com/JMURv/media-server/internal/helpers"
	c "github.com/JMURv/media-server/pkg/config"
	"github.com/JMURv/media-server/pkg/consts"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"path/filepath"
)

type MongoCleaner struct {
	conf *c.Config
}

func New(conf *c.Config) cleaner.Cleaner {
	return &MongoCleaner{
		conf: conf,
	}
}

func (c *MongoCleaner) Clean(ctx context.Context) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(c.conf.Mongo.URI))
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %s\n", err)
	}
	defer client.Disconnect(ctx)

	pathsFromDB := make(map[string]struct{})
	for _, coll := range c.conf.Mongo.Collections {
		collection := client.Database(c.conf.Mongo.Name).Collection(coll)

		err = getAllFilePathsFromDB(ctx, collection, pathsFromDB)
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

func getAllFilePathsFromDB(ctx context.Context, collection *mongo.Collection, paths map[string]struct{}) error {
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return err
		}
		findPaths(doc, paths)
	}

	return nil
}

func findPaths(doc interface{}, paths map[string]struct{}) {
	switch v := doc.(type) {
	case bson.M:
		for _, value := range v {
			findPaths(value, paths)
		}
	case bson.A:
		for _, value := range v {
			findPaths(value, paths)
		}
	case string:
		if filepath.HasPrefix(v, "/uploads") {
			paths[v] = struct{}{}
		}
	}
}
