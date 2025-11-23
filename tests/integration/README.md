# Integration Tests

Интеграционные тесты для проверки сервисного слоя приложения с реальной базой данных PostgreSQL.

## Обзор

Этот набор тестов проверяет:
- Взаимодействие сервисов с базой данных
- Корректность бизнес-логики
- Обработку ошибок и граничных случаев
- Транзакционную целостность
- Конкурентный доступ к данным
- Изоляцию транзакций

## Статистика тестов

- **Всего тестов**: 21 основных тестов
- **Всего подтестов**: 68 подтестов
- **Время выполнения**: ~1.1 секунды
- **Покрытие**: Team Service, User Service, Pull Request Service, Stats Service + Transaction tests

## Структура тестов

### Service Layer Tests

1. **team_integration_test.go** (5 тестов)
   - `TestTeamService_CreateTeam` - создание команды с участниками
   - `TestTeamService_GetTeam` - получение команды, обработка несуществующих команд
   - `TestTeamService_UpdateTeam` - добавление участников, обновление информации
   - `TestTeamService_DeactivateTeam` - деактивация участников, идемпотентность
   - `TestTeamService_ConcurrentAccess` - конкурентное создание команд

2. **user_integration_test.go** (3 теста)
   - `TestUserService_SetIsActive` - активация/деактивация пользователей
   - `TestUserService_GetUserReviews` - получение ревью пользователя
   - `TestUserService_ConcurrentActivation` - конкурентное изменение статуса

3. **pr_integration_test.go** (5 тестов)
   - `TestPullRequestService_CreatePR` - создание PR с ревьюерами и без
   - `TestPullRequestService_MergePR` - слияние PR, идемпотентность
   - `TestPullRequestService_ReassignReviewer` - переназначение ревьюера
   - `TestPullRequestService_WithInactiveUsers` - создание PR с неактивными пользователями
   - `TestPullRequestService_ConcurrentOperations` - конкурентные операции

4. **stats_integration_test.go** (2 теста)
   - `TestStatsService_GetStats` - получение статистики (пустая, с данными, с неактивными)
   - `TestStatsService_Consistency` - проверка консистентности статистики

### Transaction Tests

5. **transaction_assign_test.go** - тесты транзакций назначения ревьюеров
   - Случайный выбор ревьюеров
   - Конкретные ревьюеры
   - Обработка ошибок (уже назначены, таймаут)
   - Откат при ошибках
   - Предотвращение дедлоков
   - Проверка коммита транзакций

6. **transaction_deactivate_test.go** - тесты транзакций деактивации
   - Успешная деактивация команды
   - Таймауты
   - Пустые команды
   - Несуществующие команды

7. **transaction_general_test.go** - общие тесты транзакций
   - Восстановление после паники
   - Обработка ошибок коммита
   - Изоляция конкурентных транзакций

8. **transaction_pr_test.go** - тесты транзакций PR
   - Создание PR с откатом при ошибке
   - Идемпотентное слияние
   - Предотвращение дедлоков
   - Таймауты

9. **transaction_reassign_test.go** - тесты переназначения ревьюеров
   - Успешное переназначение
   - Откат при ошибках
   - Конкурентные переназначения
   - Атомарность операций

10. **transaction_team_test.go** - тесты транзакций команд
    - Создание команды с участниками
    - Дубликаты команд
    - Откат при ошибках
    - Обновление участников
    - Конкурентные обновления

## Требования

### База данных

Тесты требуют запущенного PostgreSQL сервера. По умолчанию используются параметры:

```
Host:     localhost
Port:     5434
Database: pr_reviewer
User:     postgres
Password: postgres
SSLMode:  disable
```

Можно переопределить через переменную окружения:

```bash
export TEST_DATABASE_URL="postgres://user:pass@host:port/dbname?sslmode=disable"
```

### Схема базы данных

Перед запуском тестов убедитесь, что схема создана:

```bash
# Применить миграции
make migrate-up

# Или напрямую
migrate -path migrations -database "postgres://postgres:postgres@localhost:5434/pr_reviewer?sslmode=disable" up
```

## Запуск тестов

### Все интеграционные тесты

```bash
# Через Makefile
make test-integration

# Или напрямую
go test -v ./tests/integration/...
```

### Конкретный тест

```bash
# Один тест
go test -v ./tests/integration/ -run TestTeamService_CreateTeam

# С подтестом
go test -v ./tests/integration/ -run TestTeamService_CreateTeam/CreateNewTeam
```

### С таймаутом

```bash
go test -v ./tests/integration/... -timeout 60s
```

### С покрытием кода

```bash
go test -v ./tests/integration/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Тестовая инфраструктура

### transaction_test_helper.go

Содержит вспомогательные функции для тестов:

- `getTestDSN()` - получение строки подключения к тестовой БД
- `setupTestDB(t)` - создание подключения к БД с автоочисткой
- `setupTestServices(t)` - создание всех сервисов для тестов
- `cleanupTestData(ctx, pool)` - очистка тестовых данных
- `testID(prefix)` - генерация уникальных ID для тестов
- `setupTestTeam(t, ctx, teamSvc, userCount)` - создание тестовой команды

### Изоляция тестов

Каждый тест:
1. Создает свои уникальные данные (ID основаны на `time.Now().UnixNano()`)
2. Использует `defer cleanup()` для автоматической очистки
3. Не зависит от других тестов
4. Может запускаться параллельно (где возможно)

### Очистка данных

После каждого теста данные удаляются в правильном порядке (соблюдение FK):

```sql
DELETE FROM pr_reviewers;
DELETE FROM pull_requests;
DELETE FROM users;
DELETE FROM teams;
```

## Примеры тестов

### Тест создания команды

```go
func TestTeamService_CreateTeam(t *testing.T) {
    teamSvc, _, _, _, cleanup := setupTestServices(t)
    defer cleanup()
    
    ctx := context.Background()
    
    t.Run("CreateNewTeam", func(t *testing.T) {
        teamName := testID("team")
        users := []domain.User{
            {ID: testID("user1"), Username: "User 1", IsActive: true},
            {ID: testID("user2"), Username: "User 2", IsActive: true},
        }
        
        team := domain.NewTeam(teamName, users)
        err := teamSvc.AddTeam(ctx, team)
        
        require.NoError(t, err)
        
        // Verify team was created
        savedTeam, err := teamSvc.GetTeam(ctx, teamName)
        require.NoError(t, err)
        assert.Equal(t, teamName, savedTeam.Name)
        assert.Len(t, savedTeam.Users, 2)
    })
}
```

### Тест конкурентного доступа

```go
func TestPullRequestService_ConcurrentOperations(t *testing.T) {
    _, _, prSvc, _, cleanup := setupTestServices(t)
    defer cleanup()
    
    t.Run("ConcurrentMerge", func(t *testing.T) {
        // Создаем один PR
        prID := testID("pr")
        // ...
        
        // 5 горутин пытаются его смержить
        var wg sync.WaitGroup
        for i := 0; i < 5; i++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                _, err := prSvc.MergePR(ctx, prID)
                require.NoError(t, err) // Должно быть идемпотентно
            }()
        }
        wg.Wait()
    })
}
```

## Отладка тестов

### Включение логов

Логирование в тестах по умолчанию установлено на уровень ERROR. Для отладки можно изменить уровень в `transaction_test_helper.go`:

```go
testLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug, // или LevelInfo
}))
```

### Просмотр данных в БД

```bash
# Подключение к тестовой БД
psql "postgres://postgres:postgres@localhost:5434/pr_reviewer"

# Просмотр данных
SELECT * FROM teams;
SELECT * FROM users;
SELECT * FROM pull_requests;
SELECT * FROM pr_reviewers;
```

### Запуск с verbose

```bash
go test -v ./tests/integration/... 2>&1 | tee test.log
```

## Troubleshooting

### Ошибка подключения к БД

```
Error: failed to connect to database
```

**Решение:**
1. Проверьте, что PostgreSQL запущен: `docker-compose ps`
2. Проверьте порт: `netstat -an | grep 5434`
3. Проверьте строку подключения в `getTestDSN()`

### Схема не создана

```
Error: relation "teams" does not exist
```

**Решение:**
```bash
make migrate-up
```

### Тесты зависают

```
Test timeout after 30s
```

**Решение:**
1. Увеличьте таймаут: `-timeout 60s`
2. Проверьте дедлоки в БД:
```sql
SELECT * FROM pg_locks WHERE NOT granted;
```

### Конфликты данных

```
Error: duplicate key value violates unique constraint
```

**Решение:**
1. Убедитесь, что используется `testID()` для генерации уникальных ID
2. Проверьте, что cleanup выполняется корректно
3. Очистите БД вручную:
```sql
TRUNCATE teams, users, pull_requests, pr_reviewers CASCADE;
```

## CI/CD Integration

Для запуска в CI/CD pipeline:

```yaml
# .github/workflows/integration-tests.yml
services:
  postgres:
    image: postgres:16
    env:
      POSTGRES_DB: pr_reviewer
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - 5434:5432

steps:
  - name: Run migrations
    run: make migrate-up
    
  - name: Run integration tests
    run: make test-integration
```

## Best Practices

1. **Изоляция**: Каждый тест должен быть независимым
2. **Cleanup**: Всегда используйте `defer cleanup()`
3. **Уникальность**: Используйте `testID()` для генерации ID
4. **Контекст**: Передавайте context.Background() в тесты
5. **Ассерты**: Используйте require для критичных проверок, assert для остальных
6. **Документация**: Комментируйте сложные тесты
7. **Конкурентность**: Тестируйте конкурентный доступ где возможно
8. **Транзакции**: Проверяйте откат при ошибках

## Дальнейшее развитие

- [ ] Добавить тесты производительности (benchmark)
- [ ] Добавить тесты миграций схемы
- [ ] Добавить тесты репликации
- [ ] Добавить мок-тесты для изоляции от БД
- [ ] Добавить snapshot тесты для статистики
- [ ] Интеграция с test coverage reporting
