# Writing Custom Storage Backends

This guide explains how to implement custom storage backends for the dynamic API server. The Storage interface allows you to use any persistence layer: databases, cloud storage, distributed systems, etc.

## Storage Interface

Every storage implementation must satisfy the `Storage` interface:

```go
type Storage interface {
    // List returns all stored objects
    List() ([]any, error)
    
    // Get retrieves a single object by ID
    Get(id string) (any, error)
    
    // Create stores a new object
    // Object must have an "id" field
    Create(obj any) error
    
    // Update modifies an existing object
    Update(id string, obj any) error
    
    // Delete removes an object by ID
    Delete(id string) error
}
```

## Core Concepts

### Object Storage

Objects are stored as-is without type knowledge. The framework:
1. Receives JSON from the API
2. Unmarshals to `map[string]interface{}` or your custom type
3. Passes to your storage backend
4. Expects you to store the object
5. Your storage converts back to the original format when retrieving

### ID Extraction

Objects must have an `"id"` field (JSON tag). Your storage should:
1. Extract the ID when creating (to use as the key)
2. Return the full object when retrieving
3. Use the provided ID for updates/deletes

Example:
```go
obj := map[string]interface{}{
    "id": "user-123",
    "name": "Alice",
    "email": "alice@example.com",
}
// Your storage extracts "user-123" as the key
```

### Error Handling

Use these error patterns:
```go
// Not found errors - framework returns 404
return fmt.Errorf("not found: %s", id)

// Already exists - framework returns 400
return fmt.Errorf("already exists: %s", id)

// Other errors - framework returns 500
return fmt.Errorf("database error: %w", err)
```

### Event Publishing

If you have an EventBus attached, publish events on changes:

```go
type MyStorage struct {
    db       *sql.DB
    eventBus api.EventBus
    resource string
}

// In Create():
s.eventBus.Publish(api.Event{
    Type:      api.Added,
    Resource:  s.resource,
    Object:    obj,
    Timestamp: time.Now(),
})
```

## Example 1: SQLite Storage

SQLite is a good choice for development and simple deployments.

### Implementation

```go
package storage

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/pergus/api-server/pkg/api"
    _ "github.com/mattn/go-sqlite3"
)

type SQLiteStorage struct {
    db       *sql.DB
    table    string
    eventBus api.EventBus
    resource string
}

// NewSQLiteStorage creates a new SQLite storage backend
func NewSQLiteStorage(dbPath string, table string) (*SQLiteStorage, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Test connection
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }
    
    storage := &SQLiteStorage{
        db:    db,
        table: table,
    }
    
    // Create table if it doesn't exist
    if err := storage.createTable(); err != nil {
        return nil, fmt.Errorf("failed to create table: %w", err)
    }
    
    return storage, nil
}

// createTable initializes the storage table
func (s *SQLiteStorage) createTable() error {
    query := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s (
            id TEXT PRIMARY KEY,
            data TEXT NOT NULL,
            created_at TIMESTAMP,
            updated_at TIMESTAMP
        )
    `, s.table)
    
    _, err := s.db.Exec(query)
    return err
}

// SetEventBus attaches an event bus for publishing changes
func (s *SQLiteStorage) SetEventBus(bus api.EventBus, resource string) {
    s.eventBus = bus
    s.resource = resource
}

// List retrieves all objects
func (s *SQLiteStorage) List() ([]any, error) {
    query := fmt.Sprintf("SELECT data FROM %s", s.table)
    
    rows, err := s.db.Query(query)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()
    
    items := []any{}
    for rows.Next() {
        var data string
        if err := rows.Scan(&data); err != nil {
            return nil, fmt.Errorf("scan failed: %w", err)
        }
        
        var obj map[string]interface{}
        if err := json.Unmarshal([]byte(data), &obj); err != nil {
            return nil, fmt.Errorf("unmarshal failed: %w", err)
        }
        
        items = append(items, obj)
    }
    
    return items, rows.Err()
}

// Get retrieves a single object by ID
func (s *SQLiteStorage) Get(id string) (any, error) {
    query := fmt.Sprintf("SELECT data FROM %s WHERE id = ?", s.table)
    
    var data string
    err := s.db.QueryRow(query, id).Scan(&data)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("not found: %s", id)
    }
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    
    var obj map[string]interface{}
    if err := json.Unmarshal([]byte(data), &obj); err != nil {
        return nil, fmt.Errorf("unmarshal failed: %w", err)
    }
    
    return obj, nil
}

// Create stores a new object
func (s *SQLiteStorage) Create(obj any) error {
    // Extract ID
    id, err := extractID(obj)
    if err != nil {
        return err
    }
    
    // Check if already exists
    query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = ?", s.table)
    var count int
    if err := s.db.QueryRow(query, id).Scan(&count); err != nil {
        return fmt.Errorf("check failed: %w", err)
    }
    if count > 0 {
        return fmt.Errorf("already exists: %s", id)
    }
    
    // Store as JSON
    data, err := json.Marshal(obj)
    if err != nil {
        return fmt.Errorf("marshal failed: %w", err)
    }
    
    insertQuery := fmt.Sprintf(
        "INSERT INTO %s (id, data, created_at, updated_at) VALUES (?, ?, ?, ?)",
        s.table,
    )
    
    _, err = s.db.Exec(insertQuery, id, string(data), time.Now(), time.Now())
    if err != nil {
        return fmt.Errorf("insert failed: %w", err)
    }
    
    // Publish event
    if s.eventBus != nil {
        s.eventBus.Publish(api.Event{
            Type:      api.Added,
            Resource:  s.resource,
            Object:    obj,
            Timestamp: time.Now(),
        })
    }
    
    return nil
}

// Update modifies an existing object
func (s *SQLiteStorage) Update(id string, obj any) error {
    // Check if exists
    query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = ?", s.table)
    var count int
    if err := s.db.QueryRow(query, id).Scan(&count); err != nil {
        return fmt.Errorf("check failed: %w", err)
    }
    if count == 0 {
        return fmt.Errorf("not found: %s", id)
    }
    
    // Store as JSON
    data, err := json.Marshal(obj)
    if err != nil {
        return fmt.Errorf("marshal failed: %w", err)
    }
    
    updateQuery := fmt.Sprintf(
        "UPDATE %s SET data = ?, updated_at = ? WHERE id = ?",
        s.table,
    )
    
    _, err = s.db.Exec(updateQuery, string(data), time.Now(), id)
    if err != nil {
        return fmt.Errorf("update failed: %w", err)
    }
    
    // Publish event
    if s.eventBus != nil {
        s.eventBus.Publish(api.Event{
            Type:      api.Modified,
            Resource:  s.resource,
            Object:    obj,
            Timestamp: time.Now(),
        })
    }
    
    return nil
}

// Delete removes an object
func (s *SQLiteStorage) Delete(id string) error {
    // Get object before deleting (for event)
    getQuery := fmt.Sprintf("SELECT data FROM %s WHERE id = ?", s.table)
    var data string
    err := s.db.QueryRow(getQuery, id).Scan(&data)
    if err == sql.ErrNoRows {
        return fmt.Errorf("not found: %s", id)
    }
    if err != nil {
        return fmt.Errorf("query failed: %w", err)
    }
    
    // Delete
    deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE id = ?", s.table)
    result, err := s.db.Exec(deleteQuery, id)
    if err != nil {
        return fmt.Errorf("delete failed: %w", err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("rows affected failed: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("not found: %s", id)
    }
    
    // Publish event with last known state
    var obj map[string]interface{}
    json.Unmarshal([]byte(data), &obj)
    
    if s.eventBus != nil {
        s.eventBus.Publish(api.Event{
            Type:      api.Deleted,
            Resource:  s.resource,
            Object:    obj,
            Timestamp: time.Now(),
        })
    }
    
    return nil
}

// Helper function to extract ID from object
func extractID(obj any) (string, error) {
    data, err := json.Marshal(obj)
    if err != nil {
        return "", fmt.Errorf("marshal failed: %w", err)
    }
    
    var m map[string]interface{}
    if err := json.Unmarshal(data, &m); err != nil {
        return "", fmt.Errorf("unmarshal failed: %w", err)
    }
    
    id, ok := m["id"]
    if !ok {
        return "", fmt.Errorf("object missing 'id' field")
    }
    
    idStr := fmt.Sprintf("%v", id)
    if idStr == "" {
        return "", fmt.Errorf("id field is empty")
    }
    
    return idStr, nil
}
```

### Usage

```go
// In main.go or plugin:
storage, err := NewSQLiteStorage("./data.db", "resources")
if err != nil {
    log.Fatalf("Failed to create storage: %v", err)
}

resource := &MyResource{
    storage: storage,
}

server.RegisterResource(resource)
```

## Example 2: S3 Storage

Store objects as JSON files in S3.

### Implementation

```go
package storage

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "time"
    
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/pergus/api-server/pkg/api"
)

type S3Storage struct {
    client   *s3.Client
    bucket   string
    prefix   string
    eventBus api.EventBus
    resource string
}

// NewS3Storage creates a new S3 storage backend
func NewS3Storage(bucket string, prefix string) (*S3Storage, error) {
    cfg, err := config.LoadDefaultConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }
    
    return &S3Storage{
        client: s3.NewFromConfig(cfg),
        bucket: bucket,
        prefix: prefix,
    }, nil
}

// SetEventBus attaches an event bus
func (s *S3Storage) SetEventBus(bus api.EventBus, resource string) {
    s.eventBus = bus
    s.resource = resource
}

// keyPath constructs the S3 key for an object
func (s *S3Storage) keyPath(id string) string {
    return s.prefix + "/" + id + ".json"
}

// List retrieves all objects (limited by S3 API)
func (s *S3Storage) List() ([]any, error) {
    items := []any{}
    
    // Note: This is a simplified implementation
    // For production, implement pagination
    ctx := context.Background()
    
    paginator := s3.NewListObjectsV2Paginator(
        s.client,
        &s3.ListObjectsV2Input{
            Bucket: aws.String(s.bucket),
            Prefix: aws.String(s.prefix),
        },
    )
    
    for paginator.HasMorePages() {
        page, err := paginator.NextPage(ctx)
        if err != nil {
            return nil, fmt.Errorf("list failed: %w", err)
        }
        
        for _, obj := range page.Contents {
            if data, err := s.getObjectData(ctx, *obj.Key); err == nil {
                items = append(items, data)
            }
        }
    }
    
    return items, nil
}

// Get retrieves a single object
func (s *S3Storage) Get(id string) (any, error) {
    ctx := context.Background()
    key := s.keyPath(id)
    
    result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, fmt.Errorf("not found: %s", id)
    }
    defer result.Body.Close()
    
    data, err := io.ReadAll(result.Body)
    if err != nil {
        return nil, fmt.Errorf("read failed: %w", err)
    }
    
    var obj map[string]interface{}
    if err := json.Unmarshal(data, &obj); err != nil {
        return nil, fmt.Errorf("unmarshal failed: %w", err)
    }
    
    return obj, nil
}

// Create stores a new object
func (s *S3Storage) Create(obj any) error {
    id, err := extractID(obj)
    if err != nil {
        return err
    }
    
    // Check if exists
    ctx := context.Background()
    key := s.keyPath(id)
    
    _, err = s.client.HeadObject(ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err == nil {
        return fmt.Errorf("already exists: %s", id)
    }
    
    // Store as JSON
    data, err := json.Marshal(obj)
    if err != nil {
        return fmt.Errorf("marshal failed: %w", err)
    }
    
    _, err = s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
        Body:   bytes.NewReader(data),
    })
    if err != nil {
        return fmt.Errorf("put failed: %w", err)
    }
    
    // Publish event
    if s.eventBus != nil {
        s.eventBus.Publish(api.Event{
            Type:      api.Added,
            Resource:  s.resource,
            Object:    obj,
            Timestamp: time.Now(),
        })
    }
    
    return nil
}

// Update modifies an existing object
func (s *S3Storage) Update(id string, obj any) error {
    ctx := context.Background()
    key := s.keyPath(id)
    
    // Check if exists
    _, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return fmt.Errorf("not found: %s", id)
    }
    
    // Store as JSON
    data, err := json.Marshal(obj)
    if err != nil {
        return fmt.Errorf("marshal failed: %w", err)
    }
    
    _, err = s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
        Body:   bytes.NewReader(data),
    })
    if err != nil {
        return fmt.Errorf("put failed: %w", err)
    }
    
    // Publish event
    if s.eventBus != nil {
        s.eventBus.Publish(api.Event{
            Type:      api.Modified,
            Resource:  s.resource,
            Object:    obj,
            Timestamp: time.Now(),
        })
    }
    
    return nil
}

// Delete removes an object
func (s *S3Storage) Delete(id string) error {
    ctx := context.Background()
    key := s.keyPath(id)
    
    // Get object before deleting (for event)
    result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return fmt.Errorf("not found: %s", id)
    }
    defer result.Body.Close()
    
    data, _ := io.ReadAll(result.Body)
    
    // Delete
    _, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return fmt.Errorf("delete failed: %w", err)
    }
    
    // Publish event
    var obj map[string]interface{}
    json.Unmarshal(data, &obj)
    
    if s.eventBus != nil {
        s.eventBus.Publish(api.Event{
            Type:      api.Deleted,
            Resource:  s.resource,
            Object:    obj,
            Timestamp: time.Now(),
        })
    }
    
    return nil
}

// Helper to get object data from S3
func (s *S3Storage) getObjectData(ctx context.Context, key string) (map[string]interface{}, error) {
    result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, err
    }
    defer result.Body.Close()
    
    data, err := io.ReadAll(result.Body)
    if err != nil {
        return nil, err
    }
    
    var obj map[string]interface{}
    err = json.Unmarshal(data, &obj)
    return obj, err
}
```

## Example 3: etcd Storage

Store objects in etcd for distributed systems.

### Implementation

```go
package storage

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/pergus/api-server/pkg/api"
    "go.etcd.io/etcd/client/v3"
)

type EtcdStorage struct {
    client   *clientv3.Client
    prefix   string
    eventBus api.EventBus
    resource string
}

// NewEtcdStorage creates a new etcd storage backend
func NewEtcdStorage(endpoints []string, prefix string) (*EtcdStorage, error) {
    client, err := clientv3.New(clientv3.Config{
        Endpoints:   endpoints,
        DialTimeout: 5 * time.Second,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create etcd client: %w", err)
    }
    
    return &EtcdStorage{
        client: client,
        prefix: prefix,
    }, nil
}

// SetEventBus attaches an event bus
func (s *EtcdStorage) SetEventBus(bus api.EventBus, resource string) {
    s.eventBus = bus
    s.resource = resource
}

// keyPath constructs the etcd key for an object
func (s *EtcdStorage) keyPath(id string) string {
    return s.prefix + "/" + id
}

// List retrieves all objects with the prefix
func (s *EtcdStorage) List() ([]any, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    resp, err := s.client.Get(ctx, s.prefix, clientv3.WithPrefix())
    if err != nil {
        return nil, fmt.Errorf("list failed: %w", err)
    }
    
    items := []any{}
    for _, kv := range resp.Kvs {
        var obj map[string]interface{}
        if err := json.Unmarshal(kv.Value, &obj); err != nil {
            continue // Skip malformed entries
        }
        items = append(items, obj)
    }
    
    return items, nil
}

// Get retrieves a single object
func (s *EtcdStorage) Get(id string) (any, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    key := s.keyPath(id)
    resp, err := s.client.Get(ctx, key)
    if err != nil {
        return nil, fmt.Errorf("get failed: %w", err)
    }
    
    if resp.Count == 0 {
        return nil, fmt.Errorf("not found: %s", id)
    }
    
    var obj map[string]interface{}
    if err := json.Unmarshal(resp.Kvs[0].Value, &obj); err != nil {
        return nil, fmt.Errorf("unmarshal failed: %w", err)
    }
    
    return obj, nil
}

// Create stores a new object
func (s *EtcdStorage) Create(obj any) error {
    id, err := extractID(obj)
    if err != nil {
        return err
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    key := s.keyPath(id)
    
    // Check if exists
    resp, err := s.client.Get(ctx, key)
    if err != nil {
        return fmt.Errorf("check failed: %w", err)
    }
    if resp.Count > 0 {
        return fmt.Errorf("already exists: %s", id)
    }
    
    // Store as JSON
    data, err := json.Marshal(obj)
    if err != nil {
        return fmt.Errorf("marshal failed: %w", err)
    }
    
    _, err = s.client.Put(ctx, key, string(data))
    if err != nil {
        return fmt.Errorf("put failed: %w", err)
    }
    
    // Publish event
    if s.eventBus != nil {
        s.eventBus.Publish(api.Event{
            Type:      api.Added,
            Resource:  s.resource,
            Object:    obj,
            Timestamp: time.Now(),
        })
    }
    
    return nil
}

// Update modifies an existing object
func (s *EtcdStorage) Update(id string, obj any) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    key := s.keyPath(id)
    
    // Check if exists
    resp, err := s.client.Get(ctx, key)
    if err != nil {
        return fmt.Errorf("check failed: %w", err)
    }
    if resp.Count == 0 {
        return fmt.Errorf("not found: %s", id)
    }
    
    // Store as JSON
    data, err := json.Marshal(obj)
    if err != nil {
        return fmt.Errorf("marshal failed: %w", err)
    }
    
    _, err = s.client.Put(ctx, key, string(data))
    if err != nil {
        return fmt.Errorf("put failed: %w", err)
    }
    
    // Publish event
    if s.eventBus != nil {
        s.eventBus.Publish(api.Event{
            Type:      api.Modified,
            Resource:  s.resource,
            Object:    obj,
            Timestamp: time.Now(),
        })
    }
    
    return nil
}

// Delete removes an object
func (s *EtcdStorage) Delete(id string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    key := s.keyPath(id)
    
    // Get before deleting (for event)
    getResp, err := s.client.Get(ctx, key)
    if err != nil {
        return fmt.Errorf("get failed: %w", err)
    }
    if getResp.Count == 0 {
        return fmt.Errorf("not found: %s", id)
    }
    
    data := getResp.Kvs[0].Value
    
    // Delete
    delResp, err := s.client.Delete(ctx, key)
    if err != nil {
        return fmt.Errorf("delete failed: %w", err)
    }
    
    if delResp.Deleted == 0 {
        return fmt.Errorf("not found: %s", id)
    }
    
    // Publish event with last known state
    var obj map[string]interface{}
    json.Unmarshal(data, &obj)
    
    if s.eventBus != nil {
        s.eventBus.Publish(api.Event{
            Type:      api.Deleted,
            Resource:  s.resource,
            Object:    obj,
            Timestamp: time.Now(),
        })
    }
    
    return nil
}
```

## Using Custom Storage in Plugins

When writing a plugin with custom storage:

```go
package main

import (
    "log"
    "github.com/pergus/api-server/pkg/api"
    "github.com/pergus/api-server/pkg/plugins"
    "myapp/storage"
)

type MyResource struct {
    storage api.Storage
}

func (r *MyResource) Name() string { return "myresource" }
func (r *MyResource) NewObject() any { return map[string]interface{}{} }
func (r *MyResource) Storage() api.Storage { return r.storage }

type MyPlugin struct {
    resource *MyResource
}

func (p *MyPlugin) Name() string { return "myresource" }

func (p *MyPlugin) Register(registry api.Registry, scheme api.Scheme) error {
    // Create custom storage
    store, err := storage.NewSQLiteStorage("./data.db", "myresource")
    if err != nil {
        return err
    }
    
    p.resource.storage = store
    
    // Register resource and type
    if err := registry.Register(p.resource); err != nil {
        return err
    }
    
    if err := scheme.Register("myresource", func() any {
        return map[string]interface{}{}
    }); err != nil {
        return err
    }
    
    log.Println("[MyPlugin] Registered with SQLite storage")
    return nil
}

func (p *MyPlugin) Unregister(registry api.Registry) error {
    return registry.Unregister("myresource")
}

var Plugin plugins.Plugin = &MyPlugin{
    resource: &MyResource{},
}
```

## Storage Comparison

| Aspect | SQLite | S3 | etcd |
|--------|--------|----|----|
| Setup | Easy (single file) | Need AWS creds | Need etcd cluster |
| Scalability | Single machine | Very large | Medium |
| Cost | Free | Per-request + storage | Hosted or self |
| Latency | Low | Medium | Low |
| Durability | Good | Very high | Very high |
| Best For | Dev, testing, small | Large objects, archive | Distributed config |

## Best Practices

1. **Error Handling**
   - Return "not found: <id>" for missing objects
   - Return "already exists: <id>" for duplicates
   - Wrap other errors with context

2. **Performance**
   - Implement caching if needed
   - Batch operations where possible
   - Use appropriate timeouts

3. **Data Consistency**
   - Ensure Create prevents duplicates
   - Ensure Update checks existence
   - Publish events reliably

4. **Testing**
   ```go
   func TestMyStorage(t *testing.T) {
       store := NewMyStorage()
       
       // Test Create
       obj := map[string]interface{}{"id": "1", "name": "test"}
       if err := store.Create(obj); err != nil {
           t.Fatalf("Create failed: %v", err)
       }
       
       // Test Get
       retrieved, err := store.Get("1")
       if err != nil {
           t.Fatalf("Get failed: %v", err)
       }
       
       // Test List
       all, err := store.List()
       if err != nil || len(all) != 1 {
           t.Fatal("List failed")
       }
   }
   ```

5. **Event Publishing**
   - Always publish after successful operations
   - Include the current state in the event
   - Use proper timestamps

## Troubleshooting

### "not found" for existing objects
- Check ID extraction logic
- Verify storage is persisting data
- Check for race conditions in concurrent access

### Events not published
- Verify EventBus is set: `storage.SetEventBus(bus, resource)`
- Check eventBus is not nil before publishing
- Verify event has correct Resource field

### Performance degradation
- Monitor query latency
- Check for connection pooling issues
- Consider adding caching layer

## Next Steps

1. Choose your storage backend (SQL, S3, etcd, etc.)
2. Implement the Storage interface
3. Handle errors appropriately
4. Publish events on changes
5. Test with your resource types
6. Integrate into plugins

See the built-in MemoryStorage at `pkg/api/storage.go` for reference implementation patterns.
