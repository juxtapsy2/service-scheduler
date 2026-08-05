# Project Context: Unified Service Scheduler (Scenario A)

You are assisting in building a production-quality Appointment Scheduler application for a dealership service system.

The goal is to replace manual booking systems with a digital scheduling platform.

## Core Requirements

The system must support:

1. Resource Constrained Booking
   - A customer requests a service appointment.
   - The request includes:
     - Vehicle
     - Service type
     - Dealership
     - Desired appointment time
   - The system must assign:
     - A qualified technician
     - An available service bay

2. Real-Time Availability Check
   - Before confirming an appointment:
     - Verify the technician is qualified for the requested service.
     - Verify the technician is available for the entire service duration.
     - Verify a service bay is available for the entire service duration.
   - Prevent overlapping bookings.

3. Confirmed Appointment Record
   - Upon successful booking:
     - Persist an Appointment record.
     - Associate:
       - Customer
       - Vehicle
       - Technician
       - Service Bay
       - Dealership
       - Service Type

---

# Technology Stack

## Backend

Language:
- Go

Framework:
- Gin

Reasons:
- High performance
- Lightweight
- Excellent concurrency support
- WebSocket support

Architecture:

Follow a clean layered architecture:


Handler
|
Service
|
Repository
|
Database


Responsibilities:

Handler:
- HTTP request/response handling
- Validation
- HTTP status codes

Service:
- Business logic
- Availability checking
- Booking workflow

Repository:
- Database queries
- Persistence logic


---

## Database

Database:
- PostgreSQL

Development:
- Self-hosted using Docker Compose

Visualization:
- DBeaver

Migration approach:
- SQL migration files

Migration order:


001_install_extensions.sql
002_create_types.sql
003_create_tables.sql
004_seed_data.sql


---

# Database Design Decisions

## Important

Do NOT introduce a ResourceReservation table.

Reason:

The Appointment table already represents resource occupancy.

Availability is calculated as:


Technician Schedule
+
Service Bay Schedule
-
Existing Appointments
=
Available Time Slots


ResourceReservation would be unnecessary complexity for this scope.

---

# Database Entities

## Customer

Stores customer information.

Fields:


id UUID PK
first_name
last_name
email
phone
created_at


---

## Vehicle

A customer can own multiple vehicles.

Fields:


id UUID PK
customer_id FK
vin
make
model
year
license_plate


---

## Dealership

Stores dealership locations.

Fields:


id UUID PK
name
address


---

## ServiceType

Defines available services.

Examples:

- Oil Change
- Brake Inspection
- Engine Diagnostics

Fields:


id UUID PK
name
duration_minutes


---

## Technician

Employees who perform services.

Fields:


id UUID PK
dealership_id FK
first_name
last_name
active


---

## TechnicianQualification

Many-to-many relationship.

A technician can perform multiple services.

Fields:


technician_id FK
service_type_id FK


Primary key:


(technician_id, service_type_id)


---

## TechnicianSchedule

Represents recurring weekly availability.

IMPORTANT:
Do not store absolute timestamps.

Use:


day_of_week
start_time
end_time
schedule_type


Example:


Monday
08:00 - 12:00 WORKING

Monday
12:00 - 13:00 BREAK

Monday
13:00 - 17:00 WORKING


Fields:


id UUID PK
technician_id FK
day_of_week
start_time TIME
end_time TIME
schedule_type ENUM


---

## ServiceBay

Physical service locations.

Fields:


id UUID PK
dealership_id FK
bay_number
active


---

## Appointment

Main booking record.

Fields:


id UUID PK

customer_id FK
vehicle_id FK
dealership_id FK
service_type_id FK

technician_id FK
service_bay_id FK

start_time
end_time

status ENUM

created_at


---

# PostgreSQL ENUM Types

Create these before tables.

## appointment_status

Values:


PENDING
CONFIRMED
CANCELLED
COMPLETED


## schedule_type

Values:


WORKING
BREAK
VACATION
TRAINING


Enum creation must be idempotent:

Use:


DO BEGIN CREATE TYPE...EXCEPTION WHEN duplicate object THEN NULL; END;


---

# Seed Data Requirements

Seed only reference data.

Include:

- 1 dealership
- Multiple service bays
- Multiple service types
- Multiple technicians
- Technician qualifications
- Technician schedules

Do NOT seed:

- Customers
- Vehicles
- Appointments

because they are transactional data.

Seed scripts must be idempotent.

Use:


INSERT ... ON CONFLICT ...


---

# Booking Logic

The booking workflow should be:


User requests appointment
|
v
Validate service type
|
v
Find qualified technicians
|
v
Check technician schedule
|
v
Check existing appointments overlap
|
v
Find available service bay
|
v
Create appointment transactionally
|
v
Commit
|
v
Notify clients via WebSocket


---

# Availability Query Rules

A technician is unavailable when:


Existing appointment.start_time < requested_end
AND
Existing appointment.end_time > requested_start


Same rule applies for service bays.

Example:

Existing:


13:30 - 14:00


Requested:


13:45 - 14:15


Overlap exists.

Reject booking.

---

# Concurrency Requirements

Multiple users may attempt to book the same technician/bay.

The backend must handle race conditions.

Use:

- PostgreSQL transactions
- Proper locking strategy if required

Expected behavior:

First successful transaction gets the booking.

Others receive:

409 Conflict

Technician Sarah is already booked
from 13:30 to 14:00.
Please select another time.
Time available: <available_times>