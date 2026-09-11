-- Listening material needs a content type of its own.
--
-- V001 fixed section_content.type to seven values, none of which is audio --
-- the club had no use for it. A language centre cannot teach listening
-- without one, and storing an mp3 as DOCUMENT would leave the player with no
-- way to know what it is holding.
--
-- The constraint is located by what it governs rather than by name: Postgres
-- generated the name, and a database restored or migrated by other means may
-- carry a different one.

DO $$
DECLARE
    constraint_name TEXT;
BEGIN
    SELECT con.conname INTO constraint_name
    FROM pg_constraint con
    WHERE con.conrelid = 'section_content'::regclass
      AND con.contype = 'c'
      AND pg_get_constraintdef(con.oid) LIKE '%type%'
      AND pg_get_constraintdef(con.oid) LIKE '%TEXT%'
      AND pg_get_constraintdef(con.oid) LIKE '%ANNOUNCEMENT%'
    LIMIT 1;

    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE section_content DROP CONSTRAINT %I', constraint_name);
    END IF;
END $$;

ALTER TABLE section_content
    ADD CONSTRAINT section_content_type_check
    CHECK (type IN (
        'TEXT','VIDEO','AUDIO','DOCUMENT','IMAGE','QUIZ','FORUM','ANNOUNCEMENT'
    ));
