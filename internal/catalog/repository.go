package catalog

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateProduct(ctx context.Context, p *Product, initialStock int, actorID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.Unit == "" {
		p.Unit = "unit"
	}
	if p.Currency == "" {
		p.Currency = "INR"
	}

	if p.Category == "" {
		p.Category = "General"
	}

	queryProduct := `
		INSERT INTO products (
			id, business_id, name, sku, barcode, category, description, unit,
			sale_price_minor, cost_price_minor, currency,
			tax_rate_basis_points, reorder_level, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err = tx.Exec(ctx, queryProduct,
		p.ID,
		p.BusinessID,
		p.Name,
		p.SKU,
		p.Barcode,
		p.Category,
		p.Description,
		p.Unit,
		p.SalePriceMinor,
		p.CostPriceMinor,
		p.Currency,
		p.TaxRateBasisPoints,
		p.ReorderLevel,
		p.CreatedAt,
		p.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrSKUAlreadyExists
		}
		return fmt.Errorf("insert product: %w", err)
	}

	stockToSet := 0
	if initialStock > 0 {
		stockToSet = initialStock
	}

	queryBalance := `
		INSERT INTO inventory_balances (
			business_id, product_id, on_hand, reserved, version, updated_at
		) VALUES ($1, $2, $3, 0, 1, $4)
	`
	_, err = tx.Exec(ctx, queryBalance, p.BusinessID, p.ID, stockToSet, now)
	if err != nil {
		return fmt.Errorf("insert inventory balance: %w", err)
	}

	if stockToSet > 0 {
		queryMovement := `
			INSERT INTO stock_movements (
				id, business_id, product_id, quantity, movement_type,
				source_type, notes, created_by, occurred_at, created_at
			) VALUES ($1, $2, $3, $4, 'initial', 'manual', 'Initial stock on product creation', $5, $6, $6)
		`
		var actorPtr *string
		if actorID != "" {
			actorPtr = &actorID
		}
		_, err = tx.Exec(ctx, queryMovement,
			uuid.NewString(),
			p.BusinessID,
			p.ID,
			stockToSet,
			actorPtr,
			now,
		)
		if err != nil {
			return fmt.Errorf("insert initial stock movement: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit product transaction: %w", err)
	}

	p.OnHand = stockToSet
	p.Reserved = 0
	return nil
}

func (r *Repository) GetProductByID(ctx context.Context, businessID, productID string) (*Product, error) {
	query := `
		SELECT
			p.id, p.business_id, p.name, p.sku, p.barcode, p.category, p.description, p.unit,
			p.sale_price_minor, p.cost_price_minor, p.currency,
			p.tax_rate_basis_points, p.reorder_level,
			COALESCE(ib.on_hand, 0), COALESCE(ib.reserved, 0),
			p.archived_at, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN inventory_balances ib ON ib.product_id = p.id AND ib.business_id = p.business_id
		WHERE p.id = $1 AND p.business_id = $2 AND p.archived_at IS NULL
	`
	var p Product
	err := r.pool.QueryRow(ctx, query, productID, businessID).Scan(
		&p.ID,
		&p.BusinessID,
		&p.Name,
		&p.SKU,
		&p.Barcode,
		&p.Category,
		&p.Description,
		&p.Unit,
		&p.SalePriceMinor,
		&p.CostPriceMinor,
		&p.Currency,
		&p.TaxRateBasisPoints,
		&p.ReorderLevel,
		&p.OnHand,
		&p.Reserved,
		&p.ArchivedAt,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("get product by id: %w", err)
	}

	return &p, nil
}

func (r *Repository) UpdateProduct(ctx context.Context, businessID, productID string, input UpdateProductInput) (*Product, error) {
	existing, err := r.GetProductByID(ctx, businessID, productID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, ErrInvalidProductData
		}
		existing.Name = trimmed
	}
	if input.Barcode != nil {
		trimmed := strings.TrimSpace(*input.Barcode)
		if trimmed == "" {
			existing.Barcode = nil
		} else {
			existing.Barcode = &trimmed
		}
	}
	if input.Category != nil {
		trimmed := strings.TrimSpace(*input.Category)
		if trimmed != "" {
			existing.Category = trimmed
		}
	}
	if input.Description != nil {
		existing.Description = input.Description
	}
	if input.Unit != nil && strings.TrimSpace(*input.Unit) != "" {
		existing.Unit = strings.TrimSpace(*input.Unit)
	}
	if input.SalePriceMinor != nil {
		existing.SalePriceMinor = *input.SalePriceMinor
	}
	if input.CostPriceMinor != nil {
		existing.CostPriceMinor = *input.CostPriceMinor
	}
	if input.TaxRateBasisPoints != nil {
		existing.TaxRateBasisPoints = *input.TaxRateBasisPoints
	}
	if input.ReorderLevel != nil {
		existing.ReorderLevel = *input.ReorderLevel
	}
	existing.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE products
		SET name = $1, barcode = $2, category = $3, description = $4, unit = $5, sale_price_minor = $6,
		    cost_price_minor = $7, tax_rate_basis_points = $8, reorder_level = $9, updated_at = $10
		WHERE id = $11 AND business_id = $12 AND archived_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query,
		existing.Name,
		existing.Barcode,
		existing.Category,
		existing.Description,
		existing.Unit,
		existing.SalePriceMinor,
		existing.CostPriceMinor,
		existing.TaxRateBasisPoints,
		existing.ReorderLevel,
		existing.UpdatedAt,
		productID,
		businessID,
	)
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrProductNotFound
	}

	return existing, nil
}

func (r *Repository) ArchiveProduct(ctx context.Context, businessID, productID string) error {
	now := time.Now().UTC()
	query := `
		UPDATE products
		SET archived_at = $1, updated_at = $1
		WHERE id = $2 AND business_id = $3 AND archived_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query, now, productID, businessID)
	if err != nil {
		return fmt.Errorf("archive product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}

	return nil
}

func (r *Repository) ListProducts(ctx context.Context, businessID string, q ListProductsQuery) ([]Product, string, error) {
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	var cursorCreatedAt time.Time
	var cursorID string
	if q.Cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(q.Cursor)
		if err == nil {
			parts := strings.Split(string(decoded), "#")
			if len(parts) == 2 {
				if parsedTime, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
					cursorCreatedAt = parsedTime
					cursorID = parts[1]
				}
			}
		}
	}

	args := []any{businessID}
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT
			p.id, p.business_id, p.name, p.sku, p.barcode, p.category, p.description, p.unit,
			p.sale_price_minor, p.cost_price_minor, p.currency,
			p.tax_rate_basis_points, p.reorder_level,
			COALESCE(ib.on_hand, 0), COALESCE(ib.reserved, 0),
			p.archived_at, p.created_at, p.updated_at
		FROM products p
		LEFT JOIN inventory_balances ib ON ib.product_id = p.id AND ib.business_id = p.business_id
		WHERE p.business_id = $1 AND p.archived_at IS NULL
	`)

	if strings.TrimSpace(q.Query) != "" {
		searchTerm := "%" + strings.TrimSpace(q.Query) + "%"
		args = append(args, searchTerm)
		paramIndex := len(args)
		queryBuilder.WriteString(fmt.Sprintf(" AND (p.name ILIKE $%d OR p.sku ILIKE $%d OR p.category ILIKE $%d)", paramIndex, paramIndex, paramIndex))
	}

	if !cursorCreatedAt.IsZero() && cursorID != "" {
		args = append(args, cursorCreatedAt, cursorID)
		paramIndex := len(args)
		queryBuilder.WriteString(fmt.Sprintf(" AND (p.created_at, p.id) < ($%d, $%d)", paramIndex-1, paramIndex))
	}

	args = append(args, limit+1)
	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY p.created_at DESC, p.id DESC LIMIT $%d", len(args)))

	rows, err := r.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, "", fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	items := make([]Product, 0, limit)
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.ID,
			&p.BusinessID,
			&p.Name,
			&p.SKU,
			&p.Barcode,
			&p.Category,
			&p.Description,
			&p.Unit,
			&p.SalePriceMinor,
			&p.CostPriceMinor,
			&p.Currency,
			&p.TaxRateBasisPoints,
			&p.ReorderLevel,
			&p.OnHand,
			&p.Reserved,
			&p.ArchivedAt,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan product: %w", err)
		}
		items = append(items, p)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate products: %w", err)
	}

	nextCursor := ""
	if len(items) > limit {
		lastItem := items[limit-1]
		items = items[:limit]
		cursorRaw := fmt.Sprintf("%s#%s", lastItem.CreatedAt.Format(time.RFC3339Nano), lastItem.ID)
		nextCursor = base64.StdEncoding.EncodeToString([]byte(cursorRaw))
	}

	return items, nextCursor, nil
}

func (r *Repository) ListInventory(ctx context.Context, businessID string, q ListInventoryQuery) ([]ProductInventoryItem, string, error) {
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	var cursorCreatedAt time.Time
	var cursorID string
	if q.Cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(q.Cursor)
		if err == nil {
			parts := strings.Split(string(decoded), "#")
			if len(parts) == 2 {
				if parsedTime, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
					cursorCreatedAt = parsedTime
					cursorID = parts[1]
				}
			}
		}
	}

	args := []any{businessID}
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT
			p.id, p.name, p.sku, p.unit, p.reorder_level,
			COALESCE(ib.on_hand, 0), COALESCE(ib.reserved, 0),
			p.created_at
		FROM products p
		LEFT JOIN inventory_balances ib ON ib.product_id = p.id AND ib.business_id = p.business_id
		WHERE p.business_id = $1 AND p.archived_at IS NULL
	`)

	if strings.TrimSpace(q.Query) != "" {
		searchTerm := "%" + strings.TrimSpace(q.Query) + "%"
		args = append(args, searchTerm)
		paramIndex := len(args)
		queryBuilder.WriteString(fmt.Sprintf(" AND (p.name ILIKE $%d OR p.sku ILIKE $%d)", paramIndex, paramIndex))
	}

	if q.LowStockOnly {
		queryBuilder.WriteString(" AND ((COALESCE(ib.on_hand, 0) - COALESCE(ib.reserved, 0)) <= p.reorder_level)")
	}

	if !cursorCreatedAt.IsZero() && cursorID != "" {
		args = append(args, cursorCreatedAt, cursorID)
		paramIndex := len(args)
		queryBuilder.WriteString(fmt.Sprintf(" AND (p.created_at, p.id) < ($%d, $%d)", paramIndex-1, paramIndex))
	}

	args = append(args, limit+1)
	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY p.created_at DESC, p.id DESC LIMIT $%d", len(args)))

	rows, err := r.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, "", fmt.Errorf("list inventory: %w", err)
	}
	defer rows.Close()

	items := make([]ProductInventoryItem, 0, limit)
	var lastCreatedAt time.Time
	var lastID string

	for rows.Next() {
		var item ProductInventoryItem
		var pCreatedAt time.Time
		if err := rows.Scan(
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.Unit,
			&item.ReorderLevel,
			&item.OnHand,
			&item.Reserved,
			&pCreatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan inventory item: %w", err)
		}
		item.Available = item.OnHand - item.Reserved
		item.IsLowStock = item.Available <= item.ReorderLevel
		items = append(items, item)
		lastCreatedAt = pCreatedAt
		lastID = item.ProductID
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate inventory: %w", err)
	}

	nextCursor := ""
	if len(items) > limit {
		items = items[:limit]
		cursorRaw := fmt.Sprintf("%s#%s", lastCreatedAt.Format(time.RFC3339Nano), lastID)
		nextCursor = base64.StdEncoding.EncodeToString([]byte(cursorRaw))
	}

	return items, nextCursor, nil
}

func (r *Repository) RecordStockAdjustment(ctx context.Context, businessID string, input StockAdjustmentInput, actorID string) (*StockMovement, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// If idempotency key provided, check if already recorded
	if input.IdempotencyKey != nil && *input.IdempotencyKey != "" {
		var existing StockMovement
		checkQuery := `
			SELECT id, business_id, product_id, quantity, movement_type, source_type, source_id, notes, created_by, idempotency_key, occurred_at, created_at
			FROM stock_movements
			WHERE business_id = $1 AND idempotency_key = $2
		`
		err := tx.QueryRow(ctx, checkQuery, businessID, *input.IdempotencyKey).Scan(
			&existing.ID,
			&existing.BusinessID,
			&existing.ProductID,
			&existing.Quantity,
			&existing.MovementType,
			&existing.SourceType,
			&existing.SourceID,
			&existing.Notes,
			&existing.CreatedBy,
			&existing.IdempotencyKey,
			&existing.OccurredAt,
			&existing.CreatedAt,
		)
		if err == nil {
			// Idempotent replay: return recorded movement
			return &existing, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("check idempotency key: %w", err)
		}
	}

	// Lock the inventory balance row
	var onHand, reserved, version int
	lockBalanceQuery := `
		SELECT on_hand, reserved, version
		FROM inventory_balances
		WHERE business_id = $1 AND product_id = $2
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, lockBalanceQuery, businessID, input.ProductID).Scan(&onHand, &reserved, &version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Check if product exists in business
			var prodExists bool
			errCheck := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM products WHERE id = $1 AND business_id = $2 AND archived_at IS NULL)", input.ProductID, businessID).Scan(&prodExists)
			if errCheck != nil || !prodExists {
				return nil, ErrProductNotFound
			}
			// If product exists but balance row is missing, initialize it
			onHand = 0
			reserved = 0
			version = 1
			_, errInit := tx.Exec(ctx, `
				INSERT INTO inventory_balances (business_id, product_id, on_hand, reserved, version, updated_at)
				VALUES ($1, $2, 0, 0, 1, NOW())
			`, businessID, input.ProductID)
			if errInit != nil {
				return nil, fmt.Errorf("init balance: %w", errInit)
			}
		} else {
			return nil, fmt.Errorf("lock inventory balance: %w", err)
		}
	}

	newOnHand := onHand + input.Quantity
	if newOnHand < 0 {
		return nil, ErrNegativeStockNotAllowed
	}
	if newOnHand < reserved {
		return nil, ErrInsufficientStock
	}

	now := time.Now().UTC()
	updateBalanceQuery := `
		UPDATE inventory_balances
		SET on_hand = $1, version = version + 1, updated_at = $2
		WHERE business_id = $3 AND product_id = $4
	`
	_, err = tx.Exec(ctx, updateBalanceQuery, newOnHand, now, businessID, input.ProductID)
	if err != nil {
		return nil, fmt.Errorf("update inventory balance: %w", err)
	}

	movementType := input.MovementType
	if movementType == "" {
		if input.Quantity >= 0 {
			movementType = "adjustment"
		} else {
			movementType = "damaged"
		}
	}

	movementID := uuid.NewString()
	insertMovementQuery := `
		INSERT INTO stock_movements (
			id, business_id, product_id, quantity, movement_type,
			source_type, notes, created_by, idempotency_key, occurred_at, created_at
		) VALUES ($1, $2, $3, $4, $5, 'manual', $6, $7, $8, $9, $9)
	`
	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}
	_, err = tx.Exec(ctx, insertMovementQuery,
		movementID,
		businessID,
		input.ProductID,
		input.Quantity,
		movementType,
		input.Notes,
		actorPtr,
		input.IdempotencyKey,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert stock movement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit stock adjustment: %w", err)
	}

	return &StockMovement{
		ID:             movementID,
		BusinessID:     businessID,
		ProductID:      input.ProductID,
		Quantity:       input.Quantity,
		MovementType:   movementType,
		SourceType:     "manual",
		Notes:          input.Notes,
		CreatedBy:      actorPtr,
		IdempotencyKey: input.IdempotencyKey,
		OccurredAt:     now,
		CreatedAt:      now,
	}, nil
}

func (r *Repository) ListMovements(ctx context.Context, businessID, productID string, q ListMovementsQuery) ([]StockMovement, string, error) {
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	var cursorCreatedAt time.Time
	var cursorID string
	if q.Cursor != "" {
		decoded, err := base64.StdEncoding.DecodeString(q.Cursor)
		if err == nil {
			parts := strings.Split(string(decoded), "#")
			if len(parts) == 2 {
				if parsedTime, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
					cursorCreatedAt = parsedTime
					cursorID = parts[1]
				}
			}
		}
	}

	args := []any{businessID, productID}
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString(`
		SELECT
			id, business_id, product_id, quantity, movement_type,
			source_type, source_id, notes, created_by, idempotency_key,
			occurred_at, created_at
		FROM stock_movements
		WHERE business_id = $1 AND product_id = $2
	`)

	if !cursorCreatedAt.IsZero() && cursorID != "" {
		args = append(args, cursorCreatedAt, cursorID)
		paramIndex := len(args)
		queryBuilder.WriteString(fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", paramIndex-1, paramIndex))
	}

	args = append(args, limit+1)
	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", len(args)))

	rows, err := r.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, "", fmt.Errorf("list stock movements: %w", err)
	}
	defer rows.Close()

	items := make([]StockMovement, 0, limit)
	for rows.Next() {
		var m StockMovement
		if err := rows.Scan(
			&m.ID,
			&m.BusinessID,
			&m.ProductID,
			&m.Quantity,
			&m.MovementType,
			&m.SourceType,
			&m.SourceID,
			&m.Notes,
			&m.CreatedBy,
			&m.IdempotencyKey,
			&m.OccurredAt,
			&m.CreatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan stock movement: %w", err)
		}
		items = append(items, m)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterate stock movements: %w", err)
	}

	nextCursor := ""
	if len(items) > limit {
		lastItem := items[limit-1]
		items = items[:limit]
		cursorRaw := fmt.Sprintf("%s#%s", lastItem.CreatedAt.Format(time.RFC3339Nano), lastItem.ID)
		nextCursor = base64.StdEncoding.EncodeToString([]byte(cursorRaw))
	}

	return items, nextCursor, nil
}
