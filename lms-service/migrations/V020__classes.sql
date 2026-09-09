-- FR-CLS-01..04 / US-A05: a class is one cohort running a course.
--
-- The fork treated a course as both the syllabus and the group taking it, so a
-- student could enrol only once per course (enrollments has UNIQUE(course_id,
-- student_id)). A language centre runs the same syllabus several times over --
-- "IELTS 6.5 evenings" and "IELTS 6.5 mornings" share a course but differ in
-- teacher, timetable and roster. That is what this table adds: the course stays
-- the material, a class is one run of it.
--
-- enrollments is kept and becomes a projection. Adding a student to a class
-- writes the matching enrollments row, so every access check, progress figure
-- and analytics query downstream keeps working untouched. Class membership is
-- the only way in: self-enrolment is gone, because a centre places its
-- learners rather than letting them pick (FR-CLS-02 names the admin as the
-- actor).

CREATE TABLE IF NOT EXISTS classes (
    id         BIGSERIAL PRIMARY KEY,
    course_id  BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    -- FR-CLS-01 and FR-CLS-03. Nullable so a class can outlive the departure of
    -- the teacher who ran it; the roster and its results must not disappear
    -- with them.
    teacher_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    -- Beyond the FRs, kept because it is what distinguishes two classes of the
    -- same course to the people reading the list. Free text: a centre writes
    -- "Tối T2-4-6, 19:00-21:00", and no query needs to understand that.
    schedule   VARCHAR(255),
    status     VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
               CHECK (status IN ('ACTIVE', 'FINISHED', 'CANCELLED')),
    created_by BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Two classes of one course must be told apart by name, and "Lop A" against
-- "lop a" is not a distinction anyone reading a timetable would make. Case
-- folded here so the constraint agrees with the check the service performs.
CREATE UNIQUE INDEX IF NOT EXISTS idx_classes_course_name
    ON classes (course_id, lower(name));

CREATE INDEX IF NOT EXISTS idx_classes_course  ON classes(course_id);
CREATE INDEX IF NOT EXISTS idx_classes_teacher ON classes(teacher_id);
CREATE INDEX IF NOT EXISTS idx_classes_status  ON classes(status);

-- A student may belong to several classes at once (US-A05 acceptance
-- criterion), including two classes of different courses.
CREATE TABLE IF NOT EXISTS class_students (
    id         BIGSERIAL PRIMARY KEY,
    class_id   BIGINT NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    student_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    added_by   BIGINT NOT NULL REFERENCES users(id),
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (class_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_class_students_class   ON class_students(class_id);
CREATE INDEX IF NOT EXISTS idx_class_students_student ON class_students(student_id);

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_classes_updated_at'
                   AND tgrelid = 'classes'::regclass) THEN
        CREATE TRIGGER update_classes_updated_at
            BEFORE UPDATE ON classes
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END $$;
