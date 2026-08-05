DO $$
BEGIN
    CREATE TYPE appointment_status AS ENUM (
        'PENDING',
        'CONFIRMED',
        'CANCELLED',
        'COMPLETED'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE schedule_type AS ENUM (
        'WORKING',
        'BREAK',
        'VACATION',
        'TRAINING'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;