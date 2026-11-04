CREATE TABLE recovery_jobs (
    booking_id uuid PRIMARY KEY,
    action text NOT NULL CHECK (action IN ('RELEASE', 'CANCEL'))
);
