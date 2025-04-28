-- Создаем временную колонку для обратного преобразования
ALTER TABLE pets ADD COLUMN photo_urls_old TEXT;

-- Преобразуем данные обратно в строку
UPDATE pets 
SET photo_urls_old = CASE
    WHEN array_length(photo_urls, 1) IS NULL THEN ''
    WHEN array_length(photo_urls, 1) = 1 THEN photo_urls[1]
    ELSE array_to_string(photo_urls, ',')
END;

-- Удаляем новую колонку и переименовываем старую
ALTER TABLE pets DROP COLUMN photo_urls;
ALTER TABLE pets RENAME COLUMN photo_urls_old TO photo_urls; 