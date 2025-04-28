-- Сначала создаем временную колонку для безопасного преобразования
ALTER TABLE pets ADD COLUMN photo_urls_new TEXT[];

-- Преобразуем существующие данные
UPDATE pets 
SET photo_urls_new = CASE
    WHEN photo_urls IS NULL OR photo_urls = '' THEN NULL
    WHEN photo_urls LIKE '{%}' THEN photo_urls::text[]
    ELSE ARRAY[photo_urls]::text[]
END;

-- Удаляем старую колонку и переименовываем новую
ALTER TABLE pets DROP COLUMN photo_urls;
ALTER TABLE pets RENAME COLUMN photo_urls_new TO photo_urls;

-- Добавляем ограничение NOT NULL и значение по умолчанию
ALTER TABLE pets ALTER COLUMN photo_urls SET DEFAULT '{}'::text[];
UPDATE pets SET photo_urls = '{}'::text[] WHERE photo_urls IS NULL;
ALTER TABLE pets ALTER COLUMN photo_urls SET NOT NULL; 