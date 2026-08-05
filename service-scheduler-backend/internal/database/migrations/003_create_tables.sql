CREATE TABLE IF NOT EXISTS dealership (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    address TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS customer (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    phone TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vehicle (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID NOT NULL REFERENCES customer(id),
    vin TEXT UNIQUE NOT NULL,
    make TEXT NOT NULL,
    model TEXT NOT NULL,
    year INT NOT NULL,
    license_plate TEXT UNIQUE
);

CREATE TABLE IF NOT EXISTS service_type (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    duration_minutes INT NOT NULL
);

CREATE TABLE IF NOT EXISTS technician (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    dealership_id UUID NOT NULL REFERENCES dealership(id),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS technician_qualification (
    technician_id UUID REFERENCES technician(id),
    service_type_id UUID REFERENCES service_type(id),
    PRIMARY KEY (technician_id, service_type_id)
);

CREATE TABLE IF NOT EXISTS technician_schedule (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    technician_id UUID NOT NULL REFERENCES technician(id),
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    schedule_type schedule_type NOT NULL,
    CONSTRAINT uq_technician_schedule
        UNIQUE (
            technician_id,
            day_of_week,
            start_time,
            end_time,
            schedule_type
        )
);

CREATE TABLE IF NOT EXISTS service_bay (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    dealership_id UUID NOT NULL REFERENCES dealership(id),
    bay_number TEXT NOT NULL,
    active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS appointment (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    customer_id UUID NOT NULL REFERENCES customer(id),
    vehicle_id UUID NOT NULL REFERENCES vehicle(id),
    dealership_id UUID NOT NULL REFERENCES dealership(id),

    service_type_id UUID NOT NULL REFERENCES service_type(id),

    technician_id UUID NOT NULL REFERENCES technician(id),
    service_bay_id UUID NOT NULL REFERENCES service_bay(id),

    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,

    status appointment_status NOT NULL DEFAULT 'CONFIRMED',

    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vehicle_customer
ON vehicle(customer_id);

CREATE INDEX IF NOT EXISTS idx_technician_dealership
ON technician(dealership_id);

CREATE INDEX IF NOT EXISTS idx_bay_dealership
ON service_bay(dealership_id);

CREATE INDEX IF NOT EXISTS idx_appointment_technician_time
ON appointment (
    technician_id,
    start_time,
    end_time
);

CREATE INDEX IF NOT EXISTS idx_appointment_bay_time
ON appointment (
    service_bay_id,
    start_time,
    end_time
);

CREATE INDEX IF NOT EXISTS idx_schedule_technician
ON technician_schedule(technician_id);