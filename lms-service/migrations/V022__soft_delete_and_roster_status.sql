-- Soft-delete users, and give a class roster entry its own status.
--
-- Two separate ideas that happen to share a migration:
--
--   users.deleted_at         — the person has left the centre entirely
--   class_students.status    — the person left THIS class
--
-- A learner who moves from the evening class to the morning one is DROPPED in
-- the first and ACTIVE in the second, while their user row stays untouched.

-- ── 1. Soft delete ──────────────────────────────────────────────────────────
--
-- lms_db.users is a thin mirror of the auth service's user table. Auth deletes
-- for real (deleteById), so once a person is removed this mirror holds the only
-- surviving copy of their name — and every created_by / graded_by column points
-- at it. Keeping the row is what lets a departed teacher still be credited on
-- the material they wrote.

ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_users_active ON users(id) WHERE deleted_at IS NULL;

-- The mirror is keyed by id, and auth already enforces one account per address.
-- The UNIQUE here is a second, redundant owner of that rule — and it is the one
-- that breaks: GetOrCreateUser looks a user up by id, so re-issuing an account
-- to someone who was deleted produces a new id with the same email, the insert
-- hits this constraint, and the sync fails silently. Soft delete makes that
-- permanent, since the old row now never goes away. Keep the index, drop the
-- uniqueness.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- ── 2. Roster status ────────────────────────────────────────────────────────
--
-- Removing a row still means "this person was never in this class" — an admin
-- correcting a mis-assignment. Leaving mid-course is a real event and keeps its
-- row, so the centre can still answer "which class did they attend".

ALTER TABLE class_students
    ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE';

ALTER TABLE class_students
    ADD COLUMN IF NOT EXISTS left_at TIMESTAMPTZ;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'class_students'::regclass
          AND conname = 'class_students_status_check'
    ) THEN
        ALTER TABLE class_students
            ADD CONSTRAINT class_students_status_check
            CHECK (status IN ('ACTIVE', 'DROPPED'));
    END IF;
END $$;

-- Every roster read that matters asks for one class's active members, and the
-- sibling-class check asks whether a student is still active anywhere on a
-- course. Both are served by these.
CREATE INDEX IF NOT EXISTS idx_class_students_active
    ON class_students(class_id) WHERE status = 'ACTIVE';

CREATE INDEX IF NOT EXISTS idx_class_students_student_active
    ON class_students(student_id) WHERE status = 'ACTIVE';
