-- Инициализация базы данных для Report Service

-- Создаем расширения если нужны
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Создаем индексы для производительности (GORM автоматически создаст таблицы)
-- Эти команды выполнятся после автомиграции

-- Функция для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Начальные данные для тестирования (опционально)
-- Таблицы будут созданы автоматически через GORM AutoMigrate

-- Выводим информацию о созданной базе
SELECT 'Report Service Database initialized successfully' AS status; 