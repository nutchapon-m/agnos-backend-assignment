-- User

CREATE TABLE IF NOT EXISTS users
(
    id bigint NOT NULL,
    username character varying(255) NOT NULL UNIQUE,
    password character varying(255),
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    PRIMARY KEY (id)
);
CREATE INDEX idx_users_deleted_at ON users(deleted_at)
WHERE deleted_at IS NULL;
CREATE SEQUENCE IF NOT EXISTS users_id_seq OWNED BY users.id;

-- Hospital

CREATE TABLE IF NOT EXISTS hospitals
(
    id bigint NOT NULL,
    code character varying(20) NOT NULL,
    name character varying(255),
    province_code character varying(2),
    is_active boolean DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    deleted_at timestamp with time zone,
    PRIMARY KEY (id)
);
CREATE INDEX idx_hospitals_deleted_at ON hospitals(deleted_at)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_hospitals_code
    ON hospitals (code) WHERE deleted_at IS NULL;
CREATE SEQUENCE IF NOT EXISTS hospitals_id_seq OWNED BY hospitals.id;

-- Patient

CREATE TABLE IF NOT EXISTS patients
(
    id bigint NOT NULL,
    national_id character varying(13),
    passport_no character varying(32),
    first_name_th character varying(100),
    middle_name_th character varying(100),
    last_name_th character varying(100),
    first_name_en character varying(100),
    middle_name_en character varying(100),
    last_name_en character varying(100),
    date_of_birth date,
    gender character varying(1) CHECK (gender IN ('M', 'F')),
    phone character varying(32),
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    deleted_at timestamp with time zone,
    PRIMARY KEY (id)
);
CREATE INDEX idx_patients_deleted_at ON patients(deleted_at)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_patients_national_id
    ON patients (national_id) WHERE national_id IS NOT NULL AND deleted_at IS NULL;
CREATE SEQUENCE IF NOT EXISTS patients_id_seq OWNED BY patients.id;

-- Staff

CREATE TABLE IF NOT EXISTS staffs
(
    id bigint NOT NULL,
    user_id bigint NOT NULL references users(id),
    employee_code character varying(32) NOT NULL,
    first_name character varying(32) NOT NULL,
    last_name character varying(32) NOT NULL,
    email character varying(255),
    license_no character varying(64),
    is_active boolean DEFAULT true,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    deleted_at timestamp with time zone,
    PRIMARY KEY (id)
);
CREATE UNIQUE INDEX uq_staff_employee_code
    ON staffs (employee_code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_staff_email
    ON staffs (lower(email)) WHERE deleted_at IS NULL;
CREATE INDEX idx_staffs_deleted_at ON staffs(deleted_at)
    WHERE deleted_at IS NULL;
CREATE SEQUENCE IF NOT EXISTS staffs_id_seq OWNED BY staffs.id;

-- Hospital patients

CREATE TABLE IF NOT EXISTS hospital_patients
(
    id bigint NOT NULL,
    hospital_id bigint NOT NULL references hospitals(id),
    patient_id bigint NOT NULL references patients(id),
    hn character varying(32) NOT NULL, -- Hospital Number / MRN
    status character varying(20) NOT NULL DEFAULT 'active',
    registered_at timestamp with time zone NOT NULL DEFAULT now(), 
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    deleted_at timestamp with time zone,
    PRIMARY KEY (id)  
);
CREATE UNIQUE INDEX uq_hospital_patients_hn
    ON hospital_patients (hospital_id, hn) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uq_hospital_patients_pair
    ON hospital_patients (hospital_id, patient_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_hospital_patients_deleted_at ON hospital_patients(deleted_at)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_hospital_patients_patient 
    ON hospital_patients(patient_id);
CREATE SEQUENCE IF NOT EXISTS hospital_patients_id_seq OWNED BY hospital_patients.id;

-- Hospital staffs

CREATE TABLE IF NOT EXISTS hospital_staffs
(
    id bigint NOT NULL,
    hospital_id bigint NOT NULL references hospitals(id),
    staff_id bigint NOT NULL references staffs(id),
    role character varying(32) CHECK (role IN ('doctor', 'nurse', 'registrar', 'admin')),
    is_primary boolean NOT NULL DEFAULT FALSE,
    effective_from date NOT NULL DEFAULT CURRENT_DATE,
    effective_to date,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    CONSTRAINT ck_sha_period CHECK (effective_to IS NULL OR effective_to >= effective_from),
    PRIMARY KEY (id)  
);
CREATE UNIQUE INDEX uq_sha_active
    ON hospital_staffs (staff_id, hospital_id) WHERE effective_to IS NULL;
CREATE UNIQUE INDEX uq_sha_one_primary
    ON hospital_staffs (staff_id) WHERE is_primary AND effective_to IS NULL;
CREATE INDEX ix_sha_lookup
    ON hospital_staffs (staff_id, effective_to);
CREATE SEQUENCE IF NOT EXISTS hospital_staffs_id_seq OWNED BY hospital_staffs.id;
