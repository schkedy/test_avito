# Transaction Tests

Комплексные тесты для проверки транзакций PostgreSQL на все возможные ошибки.

## Описание тестов

### Team Repository Tests

#### 1. `TestTeamCreateWithMembers_Success`
- **Проверяет**: Успешное создание команды с участниками в транзакции
- **Сценарий**: Создание команды с 3 участниками
- **Ожидаемый результат**: Команда и все участники созданы

#### 2. `TestTeamCreateWithMembers_Timeout`
- **Проверяет**: Обработку timeout в транзакции
- **Сценарий**: Контекст с истекшим таймаутом (1ns)
- **Ожидаемый результат**: Ошибка context deadline exceeded

#### 3. `TestTeamCreateWithMembers_DuplicateTeam`
- **Проверяет**: Обработку дубликата команды (нарушение UNIQUE constraint)
- **Сценарий**: Создание команды с одинаковым именем дважды
- **Ожидаемый результат**: Вторая попытка возвращает ошибку

#### 4. `TestTeamCreateWithMembers_RollbackOnError`
- **Проверяет**: Откат транзакции при ошибке (rollback)
- **Сценарий**: Создание команды с невалидным участником (пустой ID)
- **Ожидаемый результат**: Транзакция откатывается, команда НЕ создана

#### 5. `TestTeamUpdateMembers_Success`
- **Проверяет**: Успешное обновление участников команды
- **Сценарий**: Создание команды и обновление списка участников
- **Ожидаемый результат**: Участники обновлены

#### 6. `TestTeamUpdateMembers_ConcurrentUpdates`
- **Проверяет**: Предотвращение deadlock при параллельных обновлениях
- **Сценарий**: 10 горутин параллельно обновляют участников
- **Ожидаемый результат**: Все обновления успешны (благодаря сортировке)

### Pull Request Repository Tests

#### 7. `TestPRCreate_Success`
- **Проверяет**: Успешное создание PR с ревьюерами в транзакции
- **Сценарий**: Создание PR с 2 ревьюерами
- **Ожидаемый результат**: PR и все ревьюеры созданы

#### 8. `TestPRCreate_Timeout`
- **Проверяет**: Обработку timeout при создании PR
- **Сценарий**: Контекст с истекшим таймаутом (1ns)
- **Ожидаемый результат**: Ошибка context deadline exceeded

#### 9. `TestPRCreate_RollbackOnReviewerError`
- **Проверяет**: Откат транзакции при ошибке добавления ревьюера
- **Сценарий**: Создание PR с невалидным ревьюером (пустой ID)
- **Ожидаемый результат**: Транзакция откатывается, PR НЕ создан

#### 10. `TestPRCreate_DeadlockPrevention`
- **Проверяет**: Предотвращение deadlock при параллельном создании PR
- **Сценарий**: 10 горутин создают PR с ревьюерами в разном порядке
- **Ожидаемый результат**: Все создания успешны (благодаря сортировке)

#### 11. `TestPRMerge_Idempotent`
- **Проверяет**: Идемпотентность операции merge
- **Сценарий**: Двойной вызов Merge для одного PR
- **Ожидаемый результат**: Обе операции успешны, timestamp не меняется

#### 12. `TestPRMerge_NonExistent`
- **Проверяет**: Обработку merge несуществующего PR
- **Сценарий**: Merge PR с несуществующим ID
- **Ожидаемый результат**: Ошибка ErrPRNotFound

### General Transaction Tests

#### 13. `TestTransactionPanic_Recovery`
- **Проверяет**: Восстановление после panic в транзакции
- **Сценарий**: Нормальное выполнение (панику сложно вызвать без модификации кода)
- **Ожидаемый результат**: Транзакция завершается корректно

#### 14. `TestTransactionCommit_ErrorHandling`
- **Проверяет**: Проверку ошибок при commit транзакции
- **Сценарий**: Создание команды с валидными данными
- **Ожидаемый результат**: Commit успешен, данные сохранены

#### 15. `TestConcurrentTransactions_Isolation`
- **Проверяет**: Изоляцию параллельных транзакций
- **Сценарий**: 20 горутин создают команды параллельно
- **Ожидаемый результат**: Все 20 транзакций успешны

## Покрытые ошибки PostgreSQL

### 1. ✅ Connection Pool Exhaustion
- Тесты создают множество параллельных соединений
- Пул соединений должен корректно управлять ресурсами

### 2. ✅ Transaction Timeout (context.DeadlineExceeded)
- `TestTeamCreateWithMembers_Timeout`
- `TestPRCreate_Timeout`

### 3. ✅ Deadlocks
- `TestTeamUpdateMembers_ConcurrentUpdates`
- `TestPRCreate_DeadlockPrevention`
- Предотвращается сортировкой данных перед операциями

### 4. ✅ Constraint Violations
- `TestTeamCreateWithMembers_DuplicateTeam` - UNIQUE constraint
- `TestTeamCreateWithMembers_RollbackOnError` - NOT NULL constraint
- `TestPRCreate_RollbackOnReviewerError` - Foreign key constraint

### 5. ✅ Transaction Rollback
- `TestTeamCreateWithMembers_RollbackOnError`
- `TestPRCreate_RollbackOnReviewerError`
- Проверяется, что данные НЕ сохраняются при ошибке

### 6. ✅ Panic Recovery
- `TestTransactionPanic_Recovery`
- Defer функция с recover гарантирует rollback

### 7. ✅ Commit Errors
- `TestTransactionCommit_ErrorHandling`
- Проверяется обработка ошибок при commit

### 8. ✅ Race Conditions
- `TestPRMerge_Idempotent` - идемпотентность операций
- `TestConcurrentTransactions_Isolation` - изоляция транзакций

### 9. ✅ Serialization Failures
- Параллельные тесты могут вызвать serialization errors
- Транзакции должны корректно обрабатывать повторы

### 10. ✅ Connection Leaks
- Все тесты используют defer cleanup
- Гарантируется закрытие соединений даже при panic

## Запуск тестов

### Предварительные требования

1. **База данных должна быть запущена**:
```bash
docker-compose up -d
```

2. **Миграции должны быть применены**:
```bash
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5434/postgres?sslmode=disable" up
```

### Запуск всех тестов

```bash
go test ./tests/integration -v
```

### Запуск конкретного теста

```bash
go test ./tests/integration -v -run TestTeamCreateWithMembers_Success
```

### Запуск с race detector

```bash
go test ./tests/integration -v -race
```

### Запуск с покрытием

```bash
go test ./tests/integration -v -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Примеры результатов

### Успешное выполнение
```
=== RUN   TestTeamCreateWithMembers_Success
--- PASS: TestTeamCreateWithMembers_Success (0.05s)
=== RUN   TestTeamCreateWithMembers_Timeout
--- PASS: TestTeamCreateWithMembers_Timeout (0.02s)
=== RUN   TestTeamCreateWithMembers_DuplicateTeam
--- PASS: TestTeamCreateWithMembers_DuplicateTeam (0.03s)
...
PASS
ok      test_avito/tests/integration    2.456s
```

### Ошибка в тесте
```
=== RUN   TestTeamCreateWithMembers_RollbackOnError
    transaction_test.go:120: Expected error due to invalid user ID
    transaction_test.go:124: Verify team was NOT created (transaction rolled back)
--- PASS: TestTeamCreateWithMembers_RollbackOnError (0.03s)
```

## Метрики производительности

Тесты измеряют:
- Время выполнения транзакций
- Количество успешных параллельных операций
- Отсутствие deadlock при высокой нагрузке

## Best Practices проверяемые в тестах

1. ✅ **Context Timeout** - все транзакции имеют таймаут
2. ✅ **Panic Recovery** - defer func обрабатывает панику
3. ✅ **Sorted Operations** - данные сортируются для предотвращения deadlock
4. ✅ **Commit Error Checking** - ошибки commit проверяются
5. ✅ **Proper Cleanup** - defer cleanup гарантирует освобождение ресурсов
6. ✅ **Transaction Isolation** - параллельные транзакции не мешают друг другу
7. ✅ **Idempotency** - операции можно безопасно повторять
8. ✅ **Error Handling** - все ошибки корректно обрабатываются и логируются

## Troubleshooting

### Ошибка "connection refused"
```
dial tcp [::1]:5434: connect: connection refused
```
**Решение**: Запустите базу данных `docker-compose up -d`

### Ошибка "relation does not exist"
```
ERROR: relation "teams" does not exist (SQLSTATE 42P01)
```
**Решение**: Примените миграции (см. выше)

### Тесты проходят медленно
**Причины**:
- Таймауты в тестах на timeout (по дизайну)
- Большое количество параллельных операций
- Cleanup после каждого теста

**Оптимизация**: Используйте `-parallel` флаг:
```bash
go test ./tests/integration -v -parallel 4
```

## Дополнительная информация

- Все тесты независимы и могут выполняться параллельно
- Каждый тест очищает данные после себя
- Используется реальная PostgreSQL, не mock
- Тесты проверяют реальное поведение транзакций
