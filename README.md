# TSV Processor Service

Сервис для автоматической обработки TSV-файлов с данными устройств, сохранения в MongoDB и генерации PDF-отчетов.

## Возможности
- Автоматический мониторинг директории с TSV-файлами
- Асинхронная обработка через очередь задач (воркер-пул)
- Хранение данных в MongoDB
- Генерация PDF-отчетов по каждому устройству (с поддержкой кириллицы)
- Обработка ошибок с сохранением в БД и отдельную директорию
- Дедупликация - повторно не обрабатывает уже загруженные файлы
- REST API с пагинацией для получения данных по устройствам

## Быстрый старт

```
git clone https://github.com/wybin4/biocad.git
cd biocad

docker run -d \
  --name mongodb \
  -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=password \
  mongo:6.0

go run cmd/main.go
```

## API

### 1. Получение данных по устройству

```
GET /api/devices/{unit_guid}?page={number}&limit={number}
```
Параметры:

- unit_guid - GUID устройства
- page - номер страницы (по умолчанию 1)
- limit - записей на странице (по умолчанию 10, макс 100)

Пример:
```
curl "http://localhost:8080/api/devices/01749246-95f6-57db-b7c3-2ae0e8be671f?page=1&limit=10"
```
