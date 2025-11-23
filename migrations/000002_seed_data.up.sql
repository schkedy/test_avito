-- Миграция для заполнения базы данных тестовыми данными
-- Создаёт команды, пользователей и PR для демонстрации

-- Создание команд
INSERT INTO teams (name) VALUES
    ('backend'),
    ('frontend'),
    ('devops'),
    ('mobile'),
    ('data-science')
ON CONFLICT (name) DO NOTHING;

-- Создание пользователей в команде backend
INSERT INTO users (id, username, team_name, is_active) VALUES
    ('backend-alice', 'Alice Smith', 'backend', true),
    ('backend-bob', 'Bob Johnson', 'backend', true),
    ('backend-charlie', 'Charlie Brown', 'backend', true),
    ('backend-david', 'David Wilson', 'backend', true),
    ('backend-eve', 'Eve Davis', 'backend', false),
    ('backend-vladislav', 'Lachev Vladislav', 'backend', true)
ON CONFLICT (id) DO NOTHING;

-- Создание пользователей в команде frontend
INSERT INTO users (id, username, team_name, is_active) VALUES
    ('frontend-frank', 'Frank Miller', 'frontend', true),
    ('frontend-grace', 'Grace Lee', 'frontend', true),
    ('frontend-henry', 'Henry Moore', 'frontend', true),
    ('frontend-ivy', 'Ivy Taylor', 'frontend', true)
ON CONFLICT (id) DO NOTHING;

-- Создание пользователей в команде devops
INSERT INTO users (id, username, team_name, is_active) VALUES
    ('devops-jack', 'Jack Anderson', 'devops', true),
    ('devops-kate', 'Kate Thomas', 'devops', true),
    ('devops-leo', 'Leo Jackson', 'devops', true)
ON CONFLICT (id) DO NOTHING;

-- Создание пользователей в команде mobile
INSERT INTO users (id, username, team_name, is_active) VALUES
    ('mobile-mike', 'Mike White', 'mobile', true),
    ('mobile-nina', 'Nina Harris', 'mobile', true),
    ('mobile-oscar', 'Oscar Martin', 'mobile', true),
    ('mobile-pam', 'Pam Garcia', 'mobile', true)
ON CONFLICT (id) DO NOTHING;

-- Создание пользователей в команде data-science
INSERT INTO users (id, username, team_name, is_active) VALUES
    ('ds-quinn', 'Quinn Martinez', 'data-science', true),
    ('ds-rose', 'Rose Robinson', 'data-science', true),
    ('ds-sam', 'Sam Clark', 'data-science', true)
ON CONFLICT (id) DO NOTHING;

-- Создание Pull Requests
INSERT INTO pull_requests (id, name, author_id, status, created_at) VALUES
    ('pr-001', 'Add user authentication', 'backend-alice', 'OPEN', NOW() - INTERVAL '3 days'),
    ('pr-002', 'Fix database connection pool', 'backend-bob', 'OPEN', NOW() - INTERVAL '2 days'),
    ('pr-003', 'Implement REST API endpoints', 'backend-charlie', 'MERGED', NOW() - INTERVAL '5 days'),
    ('pr-004', 'Update React components', 'frontend-frank', 'OPEN', NOW() - INTERVAL '1 day'),
    ('pr-005', 'Redesign landing page', 'frontend-grace', 'OPEN', NOW() - INTERVAL '4 hours'),
    ('pr-006', 'Setup CI/CD pipeline', 'devops-jack', 'MERGED', NOW() - INTERVAL '7 days'),
    ('pr-007', 'Add Docker compose config', 'devops-kate', 'OPEN', NOW() - INTERVAL '12 hours'),
    ('pr-008', 'Implement push notifications', 'mobile-mike', 'OPEN', NOW() - INTERVAL '6 hours'),
    ('pr-009', 'Add ML model training script', 'ds-quinn', 'MERGED', NOW() - INTERVAL '10 days'),
    ('pr-010', 'Optimize query performance', 'backend-david', 'OPEN', NOW() - INTERVAL '8 hours'),
    ('pr-011', 'Make pr service', 'backend-vladislav', 'MERGED', NOW() - INTERVAL '6 days')
ON CONFLICT (id) DO NOTHING;

-- Обновление merged_at для смерженных PR
UPDATE pull_requests 
SET merged_at = created_at + INTERVAL '2 days' 
WHERE status = 'MERGED' AND merged_at IS NULL;

-- Назначение ревьюеров для открытых PR
INSERT INTO pr_reviewers (pull_request_id, reviewer_id, assigned_at) VALUES
    -- pr-001: Alice's PR -> Bob and Charlie review
    ('pr-001', 'backend-bob', NOW() - INTERVAL '3 days'),
    ('pr-001', 'backend-charlie', NOW() - INTERVAL '3 days'),
    
    -- pr-002: Bob's PR -> Alice and David review
    ('pr-002', 'backend-alice', NOW() - INTERVAL '2 days'),
    ('pr-002', 'backend-david', NOW() - INTERVAL '2 days'),
    
    -- pr-003: Charlie's PR (merged) -> Alice and Bob reviewed
    ('pr-003', 'backend-alice', NOW() - INTERVAL '5 days'),
    ('pr-003', 'backend-bob', NOW() - INTERVAL '5 days'),
    
    -- pr-011: Vladislav's PR (merged) -> Alice and Charlie reviewed
    ('pr-011', 'backend-alice', NOW() - INTERVAL '6 days'),
    ('pr-011', 'backend-charlie', NOW() - INTERVAL '6 days'),
    
    -- pr-004: Frank's PR -> Grace and Henry review
    ('pr-004', 'frontend-grace', NOW() - INTERVAL '1 day'),
    ('pr-004', 'frontend-henry', NOW() - INTERVAL '1 day'),
    
    -- pr-005: Grace's PR -> Frank and Ivy review
    ('pr-005', 'frontend-frank', NOW() - INTERVAL '4 hours'),
    ('pr-005', 'frontend-ivy', NOW() - INTERVAL '4 hours'),
    
    -- pr-006: Jack's PR (merged) -> Kate and Leo reviewed
    ('pr-006', 'devops-kate', NOW() - INTERVAL '7 days'),
    ('pr-006', 'devops-leo', NOW() - INTERVAL '7 days'),
    
    -- pr-007: Kate's PR -> Jack and Leo review
    ('pr-007', 'devops-jack', NOW() - INTERVAL '12 hours'),
    ('pr-007', 'devops-leo', NOW() - INTERVAL '12 hours'),
    
    -- pr-008: Mike's PR -> Nina and Oscar review
    ('pr-008', 'mobile-nina', NOW() - INTERVAL '6 hours'),
    ('pr-008', 'mobile-oscar', NOW() - INTERVAL '6 hours'),
    
    -- pr-009: Quinn's PR (merged) -> Rose and Sam reviewed
    ('pr-009', 'ds-rose', NOW() - INTERVAL '10 days'),
    ('pr-009', 'ds-sam', NOW() - INTERVAL '10 days'),
    
    -- pr-010: David's PR -> Alice and Charlie review
    ('pr-010', 'backend-alice', NOW() - INTERVAL '8 hours'),
    ('pr-010', 'backend-charlie', NOW() - INTERVAL '8 hours')
ON CONFLICT (pull_request_id, reviewer_id) DO NOTHING;
