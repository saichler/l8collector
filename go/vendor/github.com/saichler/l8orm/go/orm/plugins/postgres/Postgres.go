/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package postgres provides a PostgreSQL implementation of the IORM interface.
// It handles database connections, table creation, query execution, and includes
// an in-memory query cache with TTL support for optimized pagination performance.
package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	strings2 "strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8utils/go/utils/strings"
)

// cachedQuery represents a cached query result with its sorted RecKey array.
// This cache enables efficient pagination by storing the full result set's keys
// and serving page requests from memory rather than re-querying the database.
type cachedQuery struct {
	recKeys  []string            // Sorted array of record keys for the query
	stamp    int64               // Cache creation timestamp for invalidation
	lastUsed int64               // Last access time for TTL cleanup
	metadata *l8api.L8MetaData   // Query metadata (total count, etc.)
}

// touch updates the lastUsed timestamp to prevent TTL expiration.
func (cq *cachedQuery) touch() {
	atomic.StoreInt64(&cq.lastUsed, time.Now().Unix())
}

// pageKeys returns the subset of record keys for the requested page.
// If limit is 0 or negative, all keys are returned.
func (cq *cachedQuery) pageKeys(page, limit int32) []string {
	if limit <= 0 {
		return cq.recKeys
	}
	start := int(page * limit)
	if start >= len(cq.recKeys) {
		return []string{}
	}
	end := start + int(limit)
	if end > len(cq.recKeys) {
		end = len(cq.recKeys)
	}
	return cq.recKeys[start:end]
}

// Postgres implements the IORM interface for PostgreSQL databases.
// It provides connection pooling, automatic table creation, query caching,
// and batch write support for efficient database operations.
type Postgres struct {
	db        *sql.DB              // Database connection pool
	verifyed  map[string]bool      // Tracks verified/created tables
	mtx       *sync.Mutex          // Protects database operations
	res       ifs.IResources       // Layer 8 resources (introspector, registry, etc.)
	batchSize int                  // Maximum elements per write batch

	// Primary index for paging - caches query results for pagination
	indexMtx      *sync.RWMutex              // Protects index cache
	indexQueries  map[string]*cachedQuery    // Query hash -> cached results
	indexStamp    int64                      // Global invalidation stamp
	indexTTL      int64                      // Cache entry TTL in seconds
	indexStopCh   chan struct{}              // Signal to stop TTL cleaner
}

// NewPostgres creates a new PostgreSQL ORM instance with the given database connection.
// It initializes the query cache with a 30-second TTL and starts a background
// goroutine to clean up expired cache entries every 10 seconds.
func NewPostgres(db *sql.DB, resourcs ifs.IResources) *Postgres {
	p := &Postgres{
		db:           db,
		verifyed:     make(map[string]bool),
		mtx:          &sync.Mutex{},
		res:          resourcs,
		batchSize:    500,
		indexMtx:     &sync.RWMutex{},
		indexQueries: make(map[string]*cachedQuery),
		indexStamp:   time.Now().Unix(),
		indexTTL:     30,
		indexStopCh:  make(chan struct{}),
	}
	go p.indexTTLCleaner()
	return p
}

// indexTTLCleaner runs in a goroutine to periodically remove expired cache entries.
// It checks every 10 seconds and removes entries that haven't been accessed
// within the TTL window.
func (this *Postgres) indexTTLCleaner() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			this.cleanExpiredQueries()
		case <-this.indexStopCh:
			return
		}
	}
}

// cleanExpiredQueries removes cache entries that have exceeded their TTL.
func (this *Postgres) cleanExpiredQueries() {
	this.indexMtx.Lock()
	defer this.indexMtx.Unlock()
	now := time.Now().Unix()
	for hash, q := range this.indexQueries {
		if now-atomic.LoadInt64(&q.lastUsed) > this.indexTTL {
			delete(this.indexQueries, hash)
		}
	}
}

// invalidateIndex marks all cached queries as stale by updating the global stamp.
// Called after write or delete operations to ensure cache consistency.
func (this *Postgres) invalidateIndex() {
	this.indexMtx.Lock()
	defer this.indexMtx.Unlock()
	this.indexStamp = time.Now().Unix()
}

// collectTables recursively collects all table names needed for a type hierarchy.
// It traverses nested struct attributes to find all related table types.
func collectTables(node *l8reflect.L8Node, tables map[string]bool) {
	tables[node.TypeName] = true
	if node.Attributes != nil {
		for _, attr := range node.Attributes {
			if attr.IsStruct {
				if common.IsTimeSeriesType(attr.TypeName) {
					continue
				}
				_, ok := tables[attr.TypeName]
				if !ok {
					collectTables(attr, tables)
				}
			}
		}
	}
}

// verifyTables ensures all required tables exist in the database.
// It checks each table in the type hierarchy and creates missing tables.
func (this *Postgres) verifyTables(rootNode *l8reflect.L8Node) error {
	tables := make(map[string]bool)
	collectTables(rootNode, tables)
	for tableName, _ := range tables {
		_, ok := this.verifyed[tableName]
		if !ok {
			err := this.verifyTable(tableName)
			if err != nil {
				return err
			}
			this.verifyed[tableName] = true
		}
	}
	return nil
}

// verifyTable checks if a table exists and creates it if not.
// Uses a test query to detect non-existent tables.
func (this *Postgres) verifyTable(tableName string) error {
	q := strings.New("select * from ", tableName, " where false;")
	_, err := this.db.Exec(q.String())
	if err != nil && strings2.Contains(err.Error(), "does not exist") {
		return this.createTable(tableName)
	}
	return err
}

// createTable generates and executes DDL to create a table for the given type.
// It creates columns for all non-struct attributes and adds a composite primary
// key (ParentKey, RecKey). Non-unique indexes are created for decorated fields.
func (this *Postgres) createTable(tableName string) error {
	q := strings.New("create table ", tableName, " (\n")
	q.Add("ParentKey text,\n")
	q.Add("RecKey text,\n")
	node, ok := this.res.Introspector().NodeByTypeName(tableName)
	nonUniqueFieldsIndex, nonUniqueErr := this.res.Introspector().Decorators().Fields(node, l8reflect.L8DecoratorType_NonUnique)

	if !ok {
		return errors.New("Cannot find node for table " + tableName)
	}
	for attrName, attr := range node.Attributes {
		if attr.IsStruct {
			continue
		}
		q.Add(attrName)
		q.Add(" ")
		q.Add(postgresTypeOf(attr))
		q.Add(",\n")
	}
	q.Add("CONSTRAINT ", tableName, "_key PRIMARY KEY (ParentKey, RecKey)\n);")
	_, err := this.db.Exec(q.String())
	if err != nil {
		return err
	}

	// Create non-unique indexes if available
	if nonUniqueErr == nil && nonUniqueFieldsIndex != nil {
		fmt.Println("Creating a none unique index for ", tableName)
		for _, fieldName := range nonUniqueFieldsIndex {
			indexQ := strings.New("CREATE INDEX ", tableName, "_", fieldName, "_idx ON ", tableName, " (", fieldName, ");")
			_, err = this.db.Exec(indexQ.String())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// postgresTypeOf maps Go types to PostgreSQL column types.
// Maps and slices are stored as text (serialized), and enums default to integer.
func postgresTypeOf(node *l8reflect.L8Node) string {
	if node.IsMap || node.IsSlice {
		return "text"
	}
	switch node.TypeName {
	case "string":
		return "text"
	case "int32":
		return "integer"
	case "int64":
		return "bigint"
	case "float64":
		return "float8"
	case "float32":
		return "real"
	case "bool":
		return "boolean"
	}
	//default to enum for now - @TODO - reflect find what is the kind
	return "integer"
}

// Close stops the TTL cleaner goroutine and closes the database connection.
func (this *Postgres) Close() error {
	close(this.indexStopCh)
	this.db.Close()
	return nil
}
