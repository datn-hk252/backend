-- Seed the English skill taxonomy for the language centre.
--
-- The skills table has existed since V015 but no migration ever populated it,
-- so before this it was empty. Without a taxonomy a question cannot be tagged
-- with a skill, and a submitted attempt cannot be broken down by skill.
--
-- Four root skills following the usual split in English teaching, each with a
-- handful of sub-skills.
--
-- V016 turned this table into a competency framework: it dropped UNIQUE(name)
-- and made `code` the identity instead, through two partial unique indexes
-- (uq_skills_global_code for framework_id IS NULL, uq_skills_framework_code
-- otherwise). These rows are global, so framework_id stays NULL and the ON
-- CONFLICT clauses repeat that index predicate — a partial index requires it.
--
-- Codes are English and stable so application code can look a skill up without
-- depending on the display name; name is Vietnamese because it is shown in the
-- interface.
--
-- difficulty is left NULL on purpose: difficulty is a property of an individual
-- question, declared in question_skills.difficulty, not of the skill itself.
--
-- Re-running the migration never duplicates rows.

INSERT INTO skills (code, name, description, competency_type, parent_skill_id) VALUES
    ('LISTENING', 'Nghe', 'Listening — hiểu tiếng Anh nói',       'SKILL', NULL),
    ('SPEAKING',  'Nói',  'Speaking — diễn đạt bằng lời',         'SKILL', NULL),
    ('READING',   'Đọc',  'Reading — hiểu văn bản viết',          'SKILL', NULL),
    ('WRITING',   'Viết', 'Writing — diễn đạt bằng văn bản',      'SKILL', NULL)
ON CONFLICT (code) WHERE framework_id IS NULL AND code IS NOT NULL DO NOTHING;

INSERT INTO skills (code, name, description, competency_type, parent_skill_id)
SELECT v.code, v.name, v.description, 'SKILL', p.id
FROM (VALUES
    -- Listening
    ('LISTENING_GIST',      'Nghe – Ý chính',        'Listening for gist — nắm nội dung tổng quát',            'LISTENING'),
    ('LISTENING_DETAIL',    'Nghe – Chi tiết',       'Listening for detail — bắt thông tin cụ thể',            'LISTENING'),
    ('LISTENING_INFERENCE', 'Nghe – Suy luận',       'Listening inference — hiểu điều không nói thẳng',        'LISTENING'),

    -- Speaking
    ('SPEAKING_PRONUNCIATION', 'Nói – Phát âm',      'Pronunciation — âm, trọng âm, ngữ điệu',                 'SPEAKING'),
    ('SPEAKING_FLUENCY',       'Nói – Trôi chảy',    'Fluency — nói liền mạch, ít ngập ngừng',                 'SPEAKING'),
    ('SPEAKING_LEXIS',         'Nói – Từ vựng',      'Lexical resource — vốn từ khi nói',                      'SPEAKING'),
    ('SPEAKING_INTERACTION',   'Nói – Tương tác',    'Interaction — đáp lời, giữ mạch hội thoại',              'SPEAKING'),

    -- Reading
    ('READING_SKIMMING',  'Đọc – Ý chính',           'Skimming — đọc lướt lấy ý chính',                        'READING'),
    ('READING_SCANNING',  'Đọc – Tìm thông tin',     'Scanning — quét tìm chi tiết cụ thể',                    'READING'),
    ('READING_INFERENCE', 'Đọc – Suy luận',          'Inference — suy ra điều văn bản hàm ý',                  'READING'),
    ('READING_VOCAB',     'Đọc – Từ vựng',           'Vocabulary in context — đoán nghĩa theo ngữ cảnh',       'READING'),

    -- Writing
    ('WRITING_COHERENCE', 'Viết – Bố cục',           'Coherence and cohesion — mạch lạc, liên kết ý',          'WRITING'),
    ('WRITING_GRAMMAR',   'Viết – Ngữ pháp',         'Grammatical range and accuracy — độ chính xác ngữ pháp', 'WRITING'),
    ('WRITING_LEXIS',     'Viết – Từ vựng',          'Lexical resource — vốn từ khi viết',                     'WRITING'),
    ('WRITING_TASK',      'Viết – Đáp ứng đề bài',   'Task achievement — trả lời đúng và đủ yêu cầu',          'WRITING')
) AS v(code, name, description, parent_code)
JOIN skills p ON p.code = v.parent_code AND p.framework_id IS NULL
ON CONFLICT (code) WHERE framework_id IS NULL AND code IS NOT NULL DO NOTHING;
