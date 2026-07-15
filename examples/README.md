# Example Files

This directory contains example JSON files for all resources in the dynamic API server.

## Built-in Resources

### Users
- `user-1.json` - Alice Johnson (active)
- `user-2.json` - Bob Smith (active)

**Kind:** User → plural: users

```bash
./apictl create -f examples/user-1.json
./apictl get users
./apictl explain users
```

### Products
- `product-1.json` - Laptop ($1299.99)
- `product-2.json` - Mouse ($29.99)

**Kind:** Product → plural: products

```bash
./apictl create -f examples/product-1.json
./apictl get products
./apictl explain products
```

### Orders
- `order-1.json` - Order from alice (shipped)
- `order-2.json` - Order from bob (processing)

**Kind:** Order → plural: orders

```bash
./apictl create -f examples/order-1.json
./apictl get orders
./apictl explain orders
```

## Custom Resource Definition (CRD)

### Invoices
- `invoice-crd.yaml` - CRD definition for Invoice resource
- `invoice-1.json` - Invoice for Acme Corp ($5000, sent)
- `invoice-2.json` - Invoice for TechCorp Ltd ($15000, paid)

**Kind:** Invoice → plural: invoices (requires CRD to be created first)

```bash
# First, create the CRD
./apictl apply -f examples/invoice-crd.yaml

# Then create invoices
./apictl create -f examples/invoice-1.json
./apictl get invoices
./apictl explain invoices
```

## JSON Format

All JSON files require a `kind` field. The client uses this to:
1. Determine the resource type (pluralize the kind)
2. Set the apiVersion in the object

### Built-in Resources
```json
{
  "kind": "User",
  "id": "alice",
  "name": "Alice Johnson",
  "email": "alice@example.com",
  "is_active": true
}
```

### CRD Resources
```json
{
  "kind": "Invoice",
  "id": "inv-001",
  "customer": "Acme Corp",
  "amount": 5000.00,
  "date": "2025-07-15",
  "status": "sent"
}
```

## Full Workflow Example

```bash
# 1. Start server
./api-server

# 2. In another terminal, list initial resources
./apictl api-resources

# 3. Create some built-in resources
./apictl create -f examples/user-1.json
./apictl create -f examples/product-1.json
./apictl create -f examples/order-1.json

# 4. List them
./apictl get users
./apictl get products
./apictl get orders

# 5. Explain their schema
./apictl explain users
./apictl explain products

# 6. Apply a CRD
./apictl apply -f examples/invoice-crd.yaml

# 7. Create CRD resources
./apictl create -f examples/invoice-1.json
./apictl create -f examples/invoice-2.json

# 8. List and explain CRD resources
./apictl get invoices
./apictl explain invoices

# 9. Delete CRD (removes all resources)
./apictl delete crd invoices.example.io
./apictl api-resources  # invoices is gone

# 10. But built-in resources are still there
./apictl get users
```

## Creating Your Own Examples

To create your own example files:

1. **Choose a kind name** (e.g., "Invoice", "Product")
   - The `kind` field becomes the resource type
   - Pluralization rules apply:
     - User → users
     - Product → products
     - Order → orders
     - Invoice → invoices

2. **Include required fields:**
   - `kind` - The resource type
   - `id` - Unique identifier
   - Any other fields specific to your resource

3. **Example:**
   ```json
   {
     "kind": "MyResource",
     "id": "my-001",
     "name": "My Example",
     "status": "active"
   }
   ```

4. **Create it:**
   ```bash
   ./apictl create -f my-resource.json
   ```

## Field Information

### User
- `id` (string) - Unique user identifier
- `name` (string) - User's full name
- `email` (string) - User's email address
- `is_active` (boolean) - Whether user is active

### Product
- `id` (string) - Product identifier
- `name` (string) - Product name
- `description` (string) - Product description
- `price` (number) - Product price in USD
- `stock` (integer) - Items in stock

### Order
- `id` (string) - Order identifier
- `customer_id` (string) - ID of customer who placed order
- `total` (number) - Order total in USD
- `status` (string) - Order status (draft, processing, shipped, delivered)
- `created_at` (string) - ISO 8601 timestamp

### Invoice (CRD)
- `id` (string) - Invoice identifier
- `customer` (string) - Customer name
- `amount` (number) - Invoice amount in USD
- `date` (string) - Invoice date (YYYY-MM-DD)
- `status` (string) - Invoice status (draft, sent, paid, overdue)

## Tips

- Use `./apictl explain <resource>` to see available fields
- All objects need a unique `id` field
- The `kind` field is required for the CLI to work
- Use ISO 8601 format for dates and timestamps
- Create a CRD before creating CRD-based resources

## See Also

- `invoice-crd.yaml` - Example CRD definition
- `QUICKSTART.md` - Quick start guide
- `TESTING.md` - Test procedures
