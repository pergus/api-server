# Quick Start Guide

Get up and running with the dynamic API server and CRDs in 5 minutes.

## Build

```bash
go build -o api-server ./cmd/api-server
go build -o apictl ./cmd/apictl
```

## Run

**Terminal 1 - Start the server:**
```bash
./api-server
```

You should see:
```
Registering built-in resources...
Registered resource: users
Registered resource: products
Registered resource: orders
Starting server on http://localhost:8080
```

## Demo

**Terminal 2 - Run the demo sequence:**

### 1. List built-in resources
```bash
./apictl api-resources
```

Output:
```
NAME
orders
products
users
```

### 2. Apply a CRD
```bash
./apictl apply -f examples/invoice-crd.yaml
```

Output:
```
CRD applied: invoices.example.io
```

### 3. Verify resource appeared
```bash
./apictl api-resources
```

Output:
```
NAME
invoices
orders
products
users
```

✓ Notice `invoices` now appears!

### 4. Create an Invoice
```bash
./apictl create -f examples/invoice-1.json
```

Output:
```
invoices created: inv-001
```

### 5. List Invoices
```bash
./apictl get invoices
```

### 6. Get specific Invoice
```bash
./apictl get invoices inv-001
```

### 7. Delete the CRD
```bash
./apictl delete crd invoices.example.io
```

Output:
```
CRD deleted: invoices.example.io
```

### 8. Verify it disappeared
```bash
./apictl api-resources
```

Output:
```
NAME
orders
products
users
```

✓ `invoices` is gone!

## Key Points

- ✓ **No server restart** - Resources appear and disappear at runtime
- ✓ **No recompilation** - Single binary serves all resources
- ✓ **Discovery-based client** - `kubectl-lite` finds resources dynamically
- ✓ **Generic handlers** - One handler works for all resource types
- ✓ **Dynamic objects** - Resources don't need compiled Go structs

## Advanced Usage

### Create multiple objects
```bash
# Create invoice-2.json
cat > invoice-2.json << 'EOF'
{
  "id": "inv-002",
  "customer": "TechCorp",
  "amount": 10000.00,
  "date": "2025-07-15",
  "status": "draft"
}
EOF

./apictl create -f invoice-2.json
./apictl get invoices
```

### API Discovery
```bash
./apictl api-versions    # List API groups
curl http://localhost:8080/apis     # Via REST
```

### Direct REST API
```bash
# Create
curl -X POST http://localhost:8080/api/invoices \
  -H "Content-Type: application/json" \
  -d '{"id":"inv-003","customer":"Corp","amount":1000}'

# List
curl http://localhost:8080/api/invoices

# Get
curl http://localhost:8080/api/invoices/inv-001

# Update
curl -X PUT http://localhost:8080/api/invoices/inv-001 \
  -H "Content-Type: application/json" \
  -d '{"customer":"Acme Corp","amount":5500}'

# Delete
curl -X DELETE http://localhost:8080/api/invoices/inv-001
```

## Example Files

The `examples/` directory contains:
- `invoice-crd.yaml` - CRD definition
- `invoice-1.json` - Sample invoice object
- `DEMO.md` - Detailed demonstration

## Documentation

- `CRD_ARCHITECTURE.md` - How the CRD system works
- `ARCHITECTURE.md` - Core framework design
- `BUILD.md` - Complete build instructions

## Troubleshooting

### Server won't start
- Make sure port 8080 is available: `lsof -i :8080`
- Run from the `api-server` directory

### Client connection refused
- Is the server running? Check `ps aux | grep server`
- Is it on port 8080? Check `netstat -an | grep 8080`

### CRD apply fails
- Check the YAML syntax
- Verify required fields: group, version, kind, plural
- Look at server logs for validation errors

### kubectl-lite get crds fails
- The CRD endpoints are `/crds`, not `/api/crds`
- Try: `curl http://localhost:8080/crds`

## What's Happening

1. **Server starts** with built-in resources (users, products, orders)
2. **Generic handlers** route all requests through one dispatcher
3. **Resources registered** in thread-safe Registry
4. **Request comes in** → lookup resource → dispatch to handler
5. **CRD registered** → creates new Resource → becomes available immediately
6. **kubectl-lite** discovers available resources via API
7. **No restart ever needed** - resources appear and disappear at runtime

This demonstrates Kubernetes-style extensibility!

## Next Steps

- Read `CRD_ARCHITECTURE.md` for deep dive
- Try creating your own CRD
- Modify `examples/invoice-crd.yaml` with different fields
- Explore the source in `pkg/api/`

---

That's it! You now have a working dynamic API server with full CRD support.
