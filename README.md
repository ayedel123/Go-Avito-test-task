## Задание выполнено на Go

Реализованы все эндпоинты 

## Запуск

Для запуска введите команду `docker compose up --build`.  
Приложение собирается через Dockerfile, а в `docker-compose` создаётся ещё и контейнер с базой данных для демонстрации работы приложения.

## Переменные окружения 

Переменные окружения задаются в файле `.env`.

## Примечание

ID организаций и пользователей имеют формат `int`, потому что так указано в схеме базы данных в задании, хотя в описании апи используются строки.