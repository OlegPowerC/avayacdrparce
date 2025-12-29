### Сборщик Avaya CDR (логов звонков) с сохранением их в базу данных MySQL.

Пароль на базу данных хранится в linux ***gnome-keyring***
Установить ***gnome-keyring*** можно например так (пример для CentOS и подобных систем)
***yum install gnome-keyring***

Так же необходим python3 и pip
Если pip не установлен его можно поставить например так:
***python3 -m ensurepip --upgrade***
Теперь наду установить бибилиотеку keyring
***pip3 install keyring***

В случае если при создании пользователя получаете ошибку ***No recommended backend was available***
можно установить следующее:
***pip3 install keyrings.alt***

#### Скрипты управления паролями

**adduser.py** - Добавление пользователя и пароля в keychain.

**deleteuser.py** - Удаление пользователя из kaychain.

#### Файл настройки

**params.json** - Необходимо указать адрес сервера MySQL, имя базы данных, имя пользователя
Параметры
dburl     - адрес сервера базы данных (IP:порт)
dbname    - имя базы данных
dbuser    - имя пользователя базы данных
calltable - имя таблицы с логом вызовов
debugmode - режим отладки (0 выключен, больще 0 включен)
smsurl    - адрес сервера отправки SMS о вызове на номер содержащиеся в таблице ***smsto***

Пример:

    {"dburl":"127.0.0.1:3306",
     "dbname":"avaya",
     "dbuser":"avaya",
     "calltable":"calls",
     "debugmode":0,
     "smsurl":"http://127.0.0.1/sendsms.php","company":"PowerC"
    }

Настройка станции:
change system-parameters cdr
На первой вкладке обязательно укажите формат даты: **day/month** и primary output format **customized**
![avayacdrsettings1.JPG](avayacdrsettings1.JPG)
На второй настройте формат таким образом:

| Data item   | Lenght |
|-------------|--------|
| Date        | 6      |
| space       | 1      |
| time        | 4      |
| space       | 1      |
| sec-dur     | 5      |
| space       | 1      |
| calling-num | 15     |
| space       | 1      |
| dialed-num  | 23     |
| space       | 1      |
| auth-code   | 13     |
| space       | 1      |
| in-trk-code | 4      |
| in-crt-id   | 3      |
| space       | 1      |
| code-dial   | 4      |
| space       | 1      |
| code-used   | 4      |
| out-crt-id  | 3      |
| space       | 1      |
| cond-code   | 1      |
| space       | 1      |
| return      | 1      |
| line-feed   | 1      |

Итого должно получится 97 символов

![avayacdrsettings.JPG](avayacdrsettings.JPG)
Просмотр состояния подключения со стороны станции:
в SAT:
status cdr-link
