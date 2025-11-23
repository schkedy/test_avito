# 🧪 Запуск тестов транзакций PostgreSQL

## Быстрый старт

### 1. Убедитесь, что база данных запущена
```bash
docker-compose up -d
```

### 2. Примените миграции (если ещё не применены)
```bash
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5434/postgres?sslmode=disable" up
```

### 3. Запустите тесты
```bash
# Все тесты транзакций
go test ./tests/integration -v -run "^Test.*Transaction|^TestTeam.*|^TestPR.*" -timeout 30s

# Или короче - все тесты в integration
go test ./tests/integration -v -timeout 30s
```

---

## Команды запуска

### Стандартный запуск
```bash
go test ./tests/integration -v
```

### С race detector (рекомендуется)
```bash
go test ./tests/integration -v -race
```

### Конкретный тест
```bash
go test ./tests/integration -v -run TestPRCreate_DeadlockPrevention
```

### С покрытием кода
```bash
go test ./tests/integration -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Параллельный запуск
```bash
go test ./tests/integration -v -parallel 4
```

---

## Список всех тестов

### Team Repository Tests (6 тестов)

1. **TestTeamCreateWithMembers_Success**
   - Успешное создание команды с участниками
   ```bash
   go test ./tests/integration -v -run TestTeamCreateWithMembers_Success
   ```

2. **TestTeamCreateWithMembers_Timeout**
   - Проверка timeout в транзакции
   ```bash
   go test ./tests/integration -v -run TestTeamCreateWithMembers_Timeout
   ```

3. **TestTeamCreateWithMembers_DuplicateTeam**
   - Обработка duplicate key constraint
   ```bash
   go test ./tests/integration -v -run TestTeamCreateWithMembers_DuplicateTeam
   ```

4. **TestTeamCreateWithMembers_RollbackOnError**
   - Откат транзакции при ошибке
   ```bash
   go test ./tests/integration -v -run TestTeamCreateWithMembers_RollbackOnError
   ```

5. **TestTeamUpdateMembers_Success**
   - Успешное обновление участников
   ```bash
   go test ./tests/integration -v -run TestTeamUpdateMembers_Success
   ```

6. **TestTeamUpdateMembers_ConcurrentUpdates**
   - Параллельные обновления (deadlock prevention)
   ```bash
   go test ./tests/integration -v -run TestTeamUpdateMembers_ConcurrentUpdates
   ```

### Pull Request Repository Tests (6 тестов)

7. **TestPRCreate_Success**
   - Успешное создание PR с ревьюерами
   ```bash
   go test ./tests/integration -v -run TestPRCreate_Success
   ```

8. **TestPRCreate_Timeout**
   - Проверка timeout при создании PR
   ```bash
   go test ./tests/integration -v -run TestPRCreate_Timeout
   ```

9. **TestPRCreate_RollbackOnReviewerError**
   - Откат при ошибке добавления ревьюера
   ```bash
   go test ./tests/integration -v -run TestPRCreate_RollbackOnReviewerError
   ```

10. **TestPRCreate_DeadlockPrevention**
    - Предотвращение deadlock при параллельном создании
    ```bash
    go test ./tests/integration -v -run TestPRCreate_DeadlockPrevention
    ```

11. **TestPRMerge_Idempotent**
    - Идемпотентность операции merge
    ```bash
    go test ./tests/integration -v -run TestPRMerge_Idempotent
    ```

12. **TestPRMerge_NonExistent**
    - Обработка merge несуществующего PR
    ```bash
    go test ./tests/integration -v -run TestPRMerge_NonExistent
    ```

### General Transaction Tests (3 теста)

13. **TestTransactionPanic_Recovery**
    - Восстановление после panic
    ```bash
    go test ./tests/integration -v -run TestTransactionPanic_Recovery
    ```

14. **TestTransactionCommit_ErrorHandling**
    - Проверка ошибок commit
    ```bash
    go test ./tests/integration -v -run TestTransactionCommit_ErrorHandling
    ```

15. **TestConcurrentTransactions_Isolation**
    - Изоляция параллельных транзакций (20 concurrent)
    ```bash
    go test ./tests/integration -v -run TestConcurrentTransactions_Isolation
    ```

---

## Группы тестов

### Только Team тесты
```bash
go test ./tests/integration -v -run "^TestTeam"
```

### Только PR тесты
```bash
go test ./tests/integration -v -run "^TestPR"
```

### Только Transaction тесты
```bash
go test ./tests/integration -v -run "^TestTransaction"
```

### Только Concurrent тесты
```bash
go test ./tests/integration -v -run "Concurrent"
```

### Только Timeout тесты
```bash
go test ./tests/integration -v -run "Timeout"
```

### Только Rollback тесты
```bash
go test ./tests/integration -v -run "Rollback"
```

---

## Ожидаемый результат

### Успешный запуск
```
=== RUN   TestTeamCreateWithMembers_Success
--- PASS: TestTeamCreateWithMembers_Success (0.03s)
=== RUN   TestTeamCreateWithMembers_Timeout
--- PASS: TestTeamCreateWithMembers_Timeout (0.02s)
...
PASS
ok      test_avito/tests/integration    0.321s
```

### С race detector
```
...
--- PASS: TestConcurrentTransactions_Isolation (0.07s)
PASS
ok      test_avito/tests/integration    1.575s
```

---

## Troubleshooting

### Ошибка: connection refused
```
Error: dial tcp [::1]:5434: connect: connection refused
```

**Решение:**
```bash
docker-compose up -d
docker-compose ps  # Проверить статус
```

### Ошибка: relation does not exist
```
Error: relation "teams" does not exist (SQLSTATE 42P01)
```

**Решение:**
```bash
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5434/postgres?sslmode=disable" up
```

### Ошибка: timeout
```
Error: panic: test timed out after 30s
```

**Решение:**
```bash
# Увеличить timeout
go test ./tests/integration -v -timeout 60s
```

### Тесты медленные
**Причина:** Race detector добавляет ~5x overhead

**Решение:**
```bash
# Без race detector
go test ./tests/integration -v

# Или параллельно
go test ./tests/integration -v -parallel 4
```

---

## CI/CD Integration

### GitHub Actions пример
```yaml
name: Integration Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: postgres
        ports:
          - 5434:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      - name: Run migrations
        run: |
          migrate -path migrations -database "postgresql://postgres:postgres@localhost:5434/postgres?sslmode=disable" up
      - name: Run tests
        run: go test ./tests/integration -v -race
```

---

## Дополнительные опции

### Verbose output
```bash
go test ./tests/integration -v -race -test.v
```

### JSON output
```bash
go test ./tests/integration -json
```

### Benchmark (если добавлены)
```bash
go test ./tests/integration -bench=. -benchmem
```

### Short mode (skip long tests)
```bash
go test ./tests/integration -short
```

---

## Метрики и coverage

### Coverage report
```bash
go test ./tests/integration -coverprofile=coverage.out
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Coverage по пакетам
```bash
go test ./... -coverprofile=coverage.out -coverpkg=./...
```

---

## Best Practices для написания новых тестов

1. **Всегда используйте cleanup**
   ```go
   pool, cleanup := setupTestDB(t)
   defer cleanup()
   ```

2. **Используйте testify assertions**
   ```go
   require.NoError(t, err)
   assert.Equal(t, expected, actual)
   ```

3. **Тестируйте edge cases**
   - Timeout
   - Duplicate
   - Not found
   - Concurrent access

4. **Проверяйте rollback**
   ```go
   // Try invalid operation
   err := repo.Create(ctx, invalid)
   assert.Error(t, err)
   
   // Verify nothing was created
   _, err = repo.Get(ctx, id)
   assert.ErrorIs(t, err, ErrNotFound)
   ```

5. **Запускайте с race detector**
   ```bash
   go test ./tests/integration -race
   ```

---

## 📚 Документация

- [TRANSACTION_TESTS_README.md](TRANSACTION_TESTS_README.md) - Полное описание тестов
- [TEST_RESULTS_SUMMARY.md](TEST_RESULTS_SUMMARY.md) - Результаты и метрики
- [../SQLC_CHEATSHEET.md](../../SQLC_CHEATSHEET.md) - Работа с sqlc

---

## 🎯 Checklist перед коммитом

- [ ] Все тесты проходят: `go test ./tests/integration -v`
- [ ] Race detector чист: `go test ./tests/integration -v -race`
- [ ] Coverage > 80%: `go test ./tests/integration -cover`
- [ ] Код собирается: `go build -o bin/server ./cmd/server`
- [ ] Линтер чист: `golangci-lint run` (если установлен)

---

**Готово!** 🚀 Все тесты документированы и готовы к использованию.
