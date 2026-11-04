CREATE TABLE airlines (
    code text PRIMARY KEY,
    name text NOT NULL
);
CREATE TABLE flights (
    id uuid PRIMARY KEY,
    flight_number text NOT NULL,
    airline_code text NOT NULL REFERENCES airlines(code),
    origin text NOT NULL CHECK (origin ~ '^[A-Z]{3}$'),
    destination text NOT NULL CHECK (destination ~ '^[A-Z]{3}$'),
    departure_at timestamptz NOT NULL,
    arrival_at timestamptz NOT NULL CHECK (arrival_at > departure_at),
    total_seats integer NOT NULL CHECK (total_seats > 0),
    available_seats integer NOT NULL CHECK (available_seats BETWEEN 0 AND total_seats),
    price_minor bigint NOT NULL CHECK (price_minor > 0),
    status text NOT NULL CHECK (status IN ('SCHEDULED', 'DEPARTED', 'CANCELLED', 'COMPLETED')),
    CHECK (origin <> destination)
);
CREATE UNIQUE INDEX flight_number_day ON flights (flight_number, ((departure_at AT TIME ZONE 'UTC')::date));
CREATE TABLE seat_reservations (
    booking_id uuid PRIMARY KEY,
    flight_id uuid NOT NULL REFERENCES flights(id),
    seat_count integer NOT NULL CHECK (seat_count > 0),
    status text NOT NULL CHECK (status IN ('ACTIVE', 'RELEASED', 'EXPIRED'))
);
