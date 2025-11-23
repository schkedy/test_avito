# Миграции базы данных

## Список миграций

### 000001_init_schema
Создает основную структуру базы данных:
- Таблица `teams` - команды
- Таблица `users` - пользователи
- Таблица `pull_requests` - pull requests
- Таблица `pr_reviewers` - назначения ревьюеров

### 000002_seed_data
Заполняет базу тестовыми данными для демонстрации:

**5 команд:**
- backend (6 пользователей, 1 неактивный)
- frontend (4 пользователя)
- devops (3 пользователя)
- mobile (4 пользователя)
- data-science (3 пользователя)

**20 пользователей** с реалистичными именами

**11 Pull Requests:**
- 7 OPEN (открытых)
- 4 MERGED (смерженных)

Каждый PR имеет:
- Автора из соответствующей команды
- 2 назначенных ревьюера
- Дату создания (от 8 часов до 10 дней назад)
- Дату merge для смерженных PR

## Применение миграций

### Автоматически при запуске
Миграции применяются автоматически при `docker-compose up`:
```bash
docker-compose up -d
```

### Вручную
Применить все миграции:
```bash
migrate -path migrations -database "postgres://postgres:postgres@localhost:5434/pr_reviewer?sslmode=disable" up
```

Откатить последнюю миграцию:
```bash
migrate -path migrations -database "postgres://postgres:postgres@localhost:5434/pr_reviewer?sslmode=disable" down 1
```

### Применить seed данные вручную
```bash
cat migrations/000002_seed_data.up.sql | docker exec -i pr_reviewer_postgres psql -U postgres -d pr_reviewer
```

### Удалить seed данные
```bash
cat migrations/000002_seed_data.down.sql | docker exec -i pr_reviewer_postgres psql -U postgres -d pr_reviewer
```

## Проверка данных

После применения `000002_seed_data` можно проверить:

```bash
# Статистика
curl -s http://localhost:8080/stats | jq .

# Команда backend
curl -s 'http://localhost:8080/team/get?team_name=backend' | jq .

# PR пользователя Alice
curl -s 'http://localhost:8080/users/getReview?user_id=backend-alice' | jq .
```

**Ожидаемая статистика:**
- total_prs: 11
- open_prs: 7
- merged_prs: 4
- total_teams: 5
- total_users: 20
- active_users: 19

**Особые пользователи:**
- `backend-vladislav` (Lachev Vladislav) - автор смерженного PR "Make pr service" (pr-011)

## Примеры использования

### Тестирование переназначения
```bash
# Alice назначена ревьюером на pr-002, можно переназначить на David или Charlie
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H 'Content-Type: application/json' \
  -d '{
    "pull_request_id": "pr-002",
    "old_user_id": "backend-alice"
  }'
```

### Тестирование merge
```bash
# Смержить открытый PR
curl -X POST http://localhost:8080/pullRequest/merge \
  -H 'Content-Type: application/json' \
  -d '{
    "pull_request_id": "pr-001"
  }'
```

### Тестирование деактивации команды
```bash
# Деактивировать всех пользователей frontend команды
curl -X POST http://localhost:8080/team/deactivate \
  -H 'Content-Type: application/json' \
  -d '{
    "team_name": "frontend"
  }'
```
