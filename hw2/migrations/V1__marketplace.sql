CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('USER', 'SELLER', 'ADMIN'))
);
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description VARCHAR(4000),
    price DECIMAL(12,2) NOT NULL CHECK (price > 0),
    stock INTEGER NOT NULL CHECK (stock >= 0),
    category VARCHAR(100) NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED')),
    seller_id UUID NOT NULL REFERENCES users,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX products_status_idx ON products(status);
CREATE TABLE promo_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(20) NOT NULL UNIQUE,
    discount_type TEXT NOT NULL CHECK (discount_type IN ('PERCENTAGE', 'FIXED_AMOUNT')),
    discount_value DECIMAL(12,2) NOT NULL CHECK (discount_value > 0),
    min_order_amount DECIMAL(12,2) NOT NULL CHECK (min_order_amount >= 0),
    max_uses INTEGER NOT NULL CHECK (max_uses > 0),
    current_uses INTEGER NOT NULL DEFAULT 0 CHECK (current_uses BETWEEN 0 AND max_uses),
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ NOT NULL CHECK (valid_until >= valid_from),
    active BOOLEAN NOT NULL DEFAULT true
);
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users,
    status TEXT NOT NULL CHECK (status IN ('CREATED', 'PAYMENT_PENDING', 'PAID', 'SHIPPED', 'COMPLETED', 'CANCELED')),
    promo_code_id UUID REFERENCES promo_codes,
    total_amount DECIMAL(12,2) NOT NULL CHECK (total_amount >= 0),
    discount_amount DECIMAL(12,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX orders_one_active_idx ON orders(user_id) WHERE status IN ('CREATED', 'PAYMENT_PENDING');
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders,
    product_id UUID NOT NULL REFERENCES products,
    quantity INTEGER NOT NULL CHECK (quantity BETWEEN 1 AND 999),
    price_at_order DECIMAL(12,2) NOT NULL CHECK (price_at_order > 0),
    UNIQUE (order_id, product_id)
);
CREATE TABLE user_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users,
    operation_type TEXT NOT NULL CHECK (operation_type IN ('CREATE_ORDER', 'UPDATE_ORDER')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX user_operations_latest_idx ON user_operations(user_id, operation_type, created_at DESC);
CREATE FUNCTION touch_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = clock_timestamp();
    RETURN NEW;
END;
$$;
CREATE TRIGGER products_updated BEFORE UPDATE ON products FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER orders_updated BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
