-- Откат миграции с тестовыми данными
-- Удаляет все тестовые данные в обратном порядке зависимостей

-- Удаление назначений ревьюеров
DELETE FROM pr_reviewers WHERE pull_request_id IN (
    'pr-001', 'pr-002', 'pr-003', 'pr-004', 'pr-005',
    'pr-006', 'pr-007', 'pr-008', 'pr-009', 'pr-010', 'pr-011'
);

-- Удаление Pull Requests
DELETE FROM pull_requests WHERE id IN (
    'pr-001', 'pr-002', 'pr-003', 'pr-004', 'pr-005',
    'pr-006', 'pr-007', 'pr-008', 'pr-009', 'pr-010', 'pr-011'
);

-- Удаление пользователей
DELETE FROM users WHERE id IN (
    'backend-alice', 'backend-bob', 'backend-charlie', 'backend-david', 'backend-eve', 'backend-vladislav',
    'frontend-frank', 'frontend-grace', 'frontend-henry', 'frontend-ivy',
    'devops-jack', 'devops-kate', 'devops-leo',
    'mobile-mike', 'mobile-nina', 'mobile-oscar', 'mobile-pam',
    'ds-quinn', 'ds-rose', 'ds-sam'
);

-- Удаление команд
DELETE FROM teams WHERE name IN (
    'backend', 'frontend', 'devops', 'mobile', 'data-science'
);
