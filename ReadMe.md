# Бот для сбора информации по аккаунту на бирже Deribit.com

Для базы данных используется Postgres:
user=postgres dbname=ExchangeBot host=127.0.0.1 port=5432 sslmode=disable

##### Imports
github.com/adampointer/go-deribit
github.com/adampointer/go-deribit/client/operations
github.com/go-telegram-bot-api/telegram-bot-api
github.com/lib/pq
github.com/urShadow/go-vk-api

### Задачи

- [X] API VK/Telegram
    - [X] Прием сообщений от юзера
    - [X] Отправка сообщений юзеру
- [X] Postgres
    - [X] Запись и хранение зарегистрированных пользователей
    - [X] Использование ключей с БД
- [ ] API Deribit
    - [X] Уведомление о движениях цены
    - [X] Информация о текущей цене
    - [ ] Информация о балансе аккаунта
        - [X] Отображение текущего баланса с конвертацией в рубли
        - [ ] Исторический график баланса
    - [X] Информация по открытым позициям на бирже

