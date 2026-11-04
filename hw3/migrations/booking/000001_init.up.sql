CREATE TABLE bookings (
    id uuid PRIMARY KEY,
    user_id text NOT NULL CHECK (length(user_id) > 0),
    flight_id uuid NOT NULL,
    passenger_name text NOT NULL CHECK (length(passenger_name) > 0),
    passenger_email text NOT NULL CHECK (length(passenger_email) > 0),
    seat_count integer NOT NULL CHECK (seat_count > 0),
    total_price_minor bigint NOT NULL CHECK (total_price_minor > 0),
    status text NOT NULL CHECK (status IN ('CONFIRMED', 'CANCELLED'))
);
CREATE INDEX bookings_user ON bookings (user_id);
