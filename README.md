# 🧐 Обозреватель Хабра
### Лента кратких пересказов лучших статей с Хабра от нейросети YandexGPT

#### Приложение доступно по адресу https://habr.observer
В приложении используются материалы сайта [habr.com](https://habr.com), краткие пересказы которых получены с помощью сервиса [300.ya.ru](https://300.ya.ru).

#### Деплой
- Установить [Docker](https://docs.docker.com/engine/install/) и [Docker Compose](https://docs.docker.com/compose/install/)
- Склонировать репозиторий: `git clone https://github.com/pltnk/habr-observer.git`
- Создать внутри `.env` файл: `cp .env_example .env`
- В нём установить пользователя и пароль для базы данных, изменив значения переменных `OBSERVER_MONGO_USER` и `OBSERVER_MONGO_PASS`
- Добавить API токен для сервиса [300.ya.ru](https://300.ya.ru), изменив значение переменной `OBSERVER_AUTH_TOKEN` \
  Чтобы получить токен, нужно нажать на `API` в левом нижнем углу главной страницы сервиса, а затем нажать кнопку `Получить токен` в правом верхнем углу
- Выполнить `docker compose up -d` из корня склонированного репозитория
- Первоначальный сбор статей может занять несколько минут, так как соблюдается rate limit для API сервиса 300.ya.ru

#### Лицензия
Проект находится под лицензией [MIT](https://choosealicense.com/licenses/mit/) — подробности в файле [LICENSE](LICENSE).
