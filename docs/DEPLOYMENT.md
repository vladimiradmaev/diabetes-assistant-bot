# Развертывание ДиаАИ (DiAI) на виртуальном сервере

## 📋 Требования к серверу

### Минимальные требования
- **ОС**: Ubuntu 20.04+ / CentOS 8+ / Debian 11+
- **RAM**: 2GB (рекомендуется 4GB)
- **Диск**: 10GB свободного места
- **CPU**: 1 vCPU (рекомендуется 2 vCPU)
- **Сеть**: Доступ в интернет

### Необходимые порты
- **5432**: PostgreSQL (только для внутренней сети)
- **6379**: Redis (только для внутренней сети)
- **22**: SSH для управления

## 🛠️ Подготовка сервера

### 1. Обновление системы
```bash
# Ubuntu/Debian
sudo apt update && sudo apt upgrade -y

# CentOS/RHEL
sudo dnf update -y
```

### 2. Установка Docker и Docker Compose
```bash
# Ubuntu/Debian
sudo apt install -y apt-transport-https ca-certificates curl gnupg lsb-release

# Добавляем официальный GPG ключ Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# Добавляем репозиторий Docker
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Устанавливаем Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Добавляем пользователя в группу docker
sudo usermod -aG docker $USER

# Перелогиниваемся или выполняем
newgrp docker
```

### 3. Установка дополнительных инструментов
```bash
sudo apt install -y git make htop nano
```

## 📦 Развертывание приложения

### 1. Клонирование репозитория
```bash
# Переходим в домашнюю папку
cd ~

# Клонируем проект
git clone https://github.com/vladimiradmaev/diabetes-helper.git
cd diabetes-helper

# Переключаемся на стабильную ветку
git checkout main
```

### 2. Настройка конфигурации

#### Создание основного .env файла
```bash
# Копируем примеры конфигурации
cp .env.example .env
cp .env.logging.example .env.logging

# Редактируем основную конфигурацию
nano .env
```

**Обязательно настройте следующие параметры в .env:**
```bash
# Токен бота от @BotFather
TELEGRAM_BOT_TOKEN=

# API ключ от Google AI Studio
GEMINI_API_KEY=

# Настройки базы данных (для Docker можно оставить по умолчанию)
DB_HOST=db
DB_PORT=1111
DB_USER=postgres
DB_PASSWORD=your_secure_password_here
DB_NAME=diabetes_helper

# Настройки Redis
REDIS_HOST=redis
REDIS_PORT=1111
```

#### Настройка логирования для продакшена
```bash
# Редактируем .env.logging
nano .env.logging
```

**Для продакшена рекомендуется:**
```bash
LOG_OUTPUT=stdout
LOG_FORMAT=json
LOG_LEVEL=info
```

### 3. Проверка конфигурации
```bash
# Проверяем валидность настроек
make validate-config
```

### 4. Сборка и запуск

#### Сборка Docker образов
```bash
# Собираем образы
make build
```

#### Запуск в продакшене
```bash
# Запускаем все сервисы
make run

# Проверяем статус
make status
```

#### Проверка логов
```bash
# Универсальная команда для просмотра логов
make logs

# Только Docker логи
make logs-docker

# Логи базы данных
make logs-db
```

## 🔧 Настройка автозапуска

### 1. Создание systemd сервиса
```bash
sudo nano /etc/systemd/system/diabetes-helper.service
```

**Содержимое файла:**
```ini
[Unit]
Description=Diabetes Helper Bot
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/home/yourusername/diabetes-helper
ExecStart=/usr/bin/make run
ExecStop=/usr/bin/make stop
TimeoutStartSec=0
User=yourusername
Group=docker

[Install]
WantedBy=multi-user.target
```

### 2. Активация сервиса
```bash
# Перезагружаем systemd
sudo systemctl daemon-reload

# Включаем автозапуск
sudo systemctl enable diabetes-helper

# Запускаем сервис
sudo systemctl start diabetes-helper

# Проверяем статус
sudo systemctl status diabetes-helper
```

## 📊 Мониторинг и обслуживание

### Команды для управления
```bash
# Просмотр статуса сервисов
make status

# Просмотр логов приложения
make logs-docker

# Перезапуск приложения
make restart

# Обновление приложения
git pull origin main
make rebuild

# Резервное копирование данных
docker run --rm -v diabetes-helper_postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/backup-$(date +%Y%m%d-%H%M%S).tar.gz /data
```

### Мониторинг ресурсов
```bash
# Использование диска контейнерами
docker system df

# Использование ресурсов контейнерами
docker stats

# Статус сервисов
systemctl status diabetes-helper

# Просмотр журналов systemd
journalctl -u diabetes-helper -f
```

## 🔒 Безопасность

### 1. Настройка файрвола
```bash
# Устанавливаем ufw
sudo apt install -y ufw

# Разрешаем SSH
sudo ufw allow ssh

# Включаем файрвол
sudo ufw enable

# Проверяем статус
sudo ufw status
```

### 2. Ограничение доступа к портам БД
В конфигурации docker-compose.yml порты БД и Redis доступны только внутри Docker-сети, что обеспечивает безопасность.

### 3. Регулярные обновления
```bash
# Создаем скрипт для обновлений
nano ~/update-bot.sh
```

**Содержимое скрипта:**
```bash
#!/bin/bash
cd ~/diabetes-helper
git pull origin main
make rebuild
echo "Bot updated at $(date)" >> ~/bot-updates.log
```

```bash
# Делаем скрипт исполняемым
chmod +x ~/update-bot.sh

# Добавляем в crontab для еженедельных обновлений
crontab -e
# Добавляем строку:
# 0 3 * * 0 /home/yourusername/update-bot.sh
```

## 🚨 Устранение неполадок

### Проблемы с запуском
```bash
# Проверяем Docker
docker --version
docker compose version

# Проверяем конфигурацию
make validate-config

# Смотрим логи ошибок
make logs
```

### Проблемы с базой данных
```bash
# Проверяем статус БД
make logs-db

# Подключаемся к БД для диагностики
docker compose exec db psql -U postgres -d diabetes_helper
```

### Проблемы с памятью
```bash
# Очищаем неиспользуемые Docker образы
docker system prune -f

# Проверяем использование места
df -h
docker system df
```

### Перезапуск с чистого листа
```bash
# Останавливаем все
make stop

# Полная очистка (УДАЛИТ ВСЕ ДАННЫЕ!)
make clean

# Запуск заново
make run
```

## 📈 Масштабирование

Для высокой нагрузки можно:

1. **Увеличить ресурсы сервера**
2. **Настроить внешний Redis**
3. **Использовать внешнюю PostgreSQL**
4. **Настроить балансировку нагрузки**

### Пример конфигурации с внешними сервисами
```bash
# В .env файле
DB_HOST=your-postgres-server.com
DB_PORT=5432
REDIS_HOST=your-redis-server.com
REDIS_PORT=6379
```

## 📞 Поддержка

При возникновении проблем:

1. Проверьте логи: `make logs`
2. Проверьте конфигурацию: `make validate-config`
3. Перезапустите сервисы: `make restart`
4. Обратитесь к разработчику с логами ошибок

---

**Готово!** Ваш ДиаАИ бот развернут и готов к работе! 🚀 