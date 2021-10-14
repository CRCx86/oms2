# Order Management System (Robot)

## Локальное окружение

1. С помощью термнала перейти в корень папки сервиса, скачанной из репозитория;
2. Выполнить команды:

```
make build     # сборка исполняемого бинарного файла
make docker    # сборка docker образа приложения
make dcup      # запуск всех контейнеров локального окружения
make dcgrayup  # запуск окружения с graylog 
```

Остановка всех контейнеров локального окружения:
```
make dcdown dcgraydown
```

Очистка локального окружения от предыдущих сборок (удаление неиспользуемых сборок и томов(volumes)):
```
make clean
```

## Интеграционные тесты
Перед запуском интеграционных тестов необходимо удостовериться что в регистре образ с текущим кодом системы, который 
нужно тестировать.
Для актуализации нужно использовать команду:
```
make build
make docker
```

Для запуска тестов в CI-режиме используется команда:
```
make inttest-ci
```
В случае успешного завершения тестов контейнеры успешно завершатся и удалятся. Если произошла ошибка, docker-compose 
возвращает код ответа не равный 0 и CI должен прерываться.
Для локального запуска тестов и отладки можно использовать команду:
```
make inttest-debug-up
```
Остановка всех контейнеров тестового окружения:
```
make inttest-debug-down
```

Запуск пересборки контейнера с интеграционными тестами и повторным запуском прохождения тестов:
```
make inttest-debug-restart-tests
```

## Трэйсинг

Трэйсы пишем в Jaeger, локально трэйсы можно посмотреть через Jaeger UI по адресу http://0.0.0.0:16686

## Настройка graylog2

 * Hostname: 127.0.0.1:9000 
 * Login/Password: admin - admin

1. В верхнем меню кликаем на [System]. 
2. В выпадающем меню [System] находим подраздел [Inputs] и кликаем на него.   
![](assets/gr_screen_1.png) 
3. В выпадающем списке (Select input) выбираем [GELF UDP].
![](assets/gr_screen_2.png)
4. В выпадающем списке (Node) выбираем текущую ноду.
![](assets/gr_screen_3.png)
5. В поле Title указываем [test] или другой текст по желанию и нажимаем кнопку [Save].
![](assets/gr_screen_4.png)
6. В списке (Local Inputs) появился локальный адаптер. Нажимаем на кнопку [Show received messages].
![](assets/gr_screen_5.png)
7. Список сообщений. 
![](assets/gr_screen_6.png)
8. Нажимаем на любое сообщение и переходим к разделу [message]. Справа от сообщения находим выпадающее меню и кликаем на [Create extractor for fields] выбираем [JSON]. 
![](assets/gr_screen_7.png)
Заполняем поле Title и нажимаем на кнопку [Save].
![](assets/gr_screen_8.png)
Done!
![](assets/gr_screen_9.png)

## Описание таблиц баз данных

### Аналоги регистров сведений
1. _InfoReg_ES - семафоры обработки событий (Event Semaphores)
2. _InfoReg_CSR - текущий шаг маршрута (Current Step Route)

### Аналоги справочников и табличный частей справочников
1. _Ref_M - Карта процессов (Map)
2. _RefVT_ME - Табличная часть событий для обработки (Map Events)
3. _Ref_E - События (Events)
4. _Ref_ET - Варианты событий (Event Types)
5. _Ref_L - Лоты (Lots)
5. _Ref_S - Лоты (Shipments)
5. _Ref_D - Лоты (Deliveries)
5. _Ref_O - Лоты (Orders)