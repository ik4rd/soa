CREATE TABLE route_versions (
    origin text NOT NULL,
    destination text NOT NULL,
    version uuid NOT NULL DEFAULT gen_random_uuid(),
    PRIMARY KEY (origin, destination)
);
INSERT INTO route_versions (origin, destination) SELECT DISTINCT origin, destination FROM flights;
CREATE FUNCTION invalidate_route() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        INSERT INTO route_versions (origin, destination) VALUES (OLD.origin, OLD.destination)
        ON CONFLICT (origin, destination) DO UPDATE SET version = gen_random_uuid();
    END IF;
    IF TG_OP <> 'DELETE' THEN
        INSERT INTO route_versions (origin, destination) VALUES (NEW.origin, NEW.destination)
        ON CONFLICT (origin, destination) DO UPDATE SET version = gen_random_uuid();
    END IF;
    RETURN NULL;
END;
$$;
CREATE TRIGGER flights_cache AFTER INSERT OR UPDATE OR DELETE ON flights FOR EACH ROW EXECUTE FUNCTION invalidate_route();
CREATE TABLE reservation_cancellations (booking_id uuid PRIMARY KEY);
CREATE INDEX flights_search ON flights (origin, destination, departure_at) WHERE status = 'SCHEDULED';
