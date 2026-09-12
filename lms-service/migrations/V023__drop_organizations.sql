-- Remove organisations from the LMS.
--
-- The fork served a federation of clubs, so a course belonged to an
-- organisation and could be hidden from everyone outside it. A language centre
-- is one organisation; the concept had no second value to take, and the whole
-- of it sat on a single seeded row named after the club.
--
-- Until now every course creation fell through to that row: the frontend stopped
-- sending org_id once the picker was removed, the membership table was left
-- empty once the auth service's club business was parked, so the code reached
-- its last resort and looked up the organisation with slug 'bdc'. Deleting that
-- row - the obvious thing to do when clearing club-era data - would have broken
-- course creation outright.

-- ── 1. Courses stop belonging to an organisation ────────────────────────────

DROP INDEX IF EXISTS idx_courses_org_visibility_status;
DROP INDEX IF EXISTS idx_courses_org_status_pub;
DROP INDEX IF EXISTS idx_courses_org_published_page;

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_org_id_fkey;
ALTER TABLE courses DROP COLUMN IF EXISTS org_id;

-- ── 2. Visibility keeps one value ───────────────────────────────────────────
--
-- ORG_ONLY meant "members of the owning organisation only", which now names
-- nobody. Any row still carrying it would fail the new constraint, so normalise
-- first - on a centre's data this touches nothing, but a migration that assumes
-- its own data is clean is a migration that fails on somebody else's machine.

UPDATE courses SET visibility = 'PUBLIC' WHERE visibility <> 'PUBLIC';

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_visibility_check;

ALTER TABLE courses
    ADD CONSTRAINT courses_visibility_check CHECK (visibility = 'PUBLIC');

-- ── 3. The competency framework lets go too ─────────────────────────────────
--
-- competency_frameworks was built to let each organisation carry its own
-- taxonomy, so it holds a foreign key into organizations which would block the
-- drop below. The table itself stays: skills.framework_id still points at it,
-- and V019's partial unique indexes are written in terms of that column. Only
-- the ownership link goes, which nothing in the service ever read.

ALTER TABLE competency_frameworks
    DROP CONSTRAINT IF EXISTS competency_frameworks_organization_id_fkey;

ALTER TABLE competency_frameworks DROP COLUMN IF EXISTS organization_id;

-- ── 4. The tables themselves ────────────────────────────────────────────────
--
-- organization_members first: it carries the foreign key into organizations.

DROP TABLE IF EXISTS organization_members;
DROP TABLE IF EXISTS organizations;
