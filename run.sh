#!/bin/bash

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Запуск ДиаАИ (DiAI) - Умный помощник для диабетиков${NC}"
echo ""

# Проверяем наличие .env файла
if [ ! -f .env ]; then
    echo -e "${RED}❌ Файл .env не найден!${NC}"
    echo -e "${YELLOW}Создайте файл .env с необходимыми переменными:${NC}"
    echo "cp .env.example .env"
    echo "# Затем отредактируйте .env файл с вашими токенами"
    exit 1
fi

# Проверяем наличие Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker не установлен!${NC}"
    exit 1
fi

# Определяем команду Docker Compose
DOCKER_COMPOSE_CMD=""
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
else
    echo -e "${RED}❌ Docker Compose не установлен!${NC}"
    echo -e "${YELLOW}Установите Docker Compose или обновите Docker до версии с встроенным Compose${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Используется: $DOCKER_COMPOSE_CMD${NC}"

# Останавливаем предыдущие контейнеры если они запущены
echo -e "${YELLOW}🛑 Остановка предыдущих контейнеров...${NC}"
$DOCKER_COMPOSE_CMD down > /dev/null 2>&1

# Собираем образы
echo -e "${YELLOW}🔨 Сборка Docker образов...${NC}"
$DOCKER_COMPOSE_CMD build

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Ошибка при сборке образов!${NC}"
    exit 1
fi

# Запускаем сервисы
echo -e "${GREEN}🚀 Запуск приложения с PostgreSQL...${NC}"
$DOCKER_COMPOSE_CMD up -d

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Ошибка при запуске сервисов!${NC}"
    exit 1
fi

# Ждем готовности базы данных
echo -e "${YELLOW}⏳ Ожидание готовности базы данных...${NC}"
timeout=60
counter=0
while ! $DOCKER_COMPOSE_CMD exec -T db pg_isready -U postgres > /dev/null 2>&1; do
    sleep 2
    counter=$((counter + 2))
    if [ $counter -ge $timeout ]; then
        echo -e "${RED}❌ Таймаут ожидания базы данных!${NC}"
        $DOCKER_COMPOSE_CMD logs db
        exit 1
    fi
done

echo -e "${GREEN}✅ База данных готова!${NC}"

# Проверяем статус приложения
sleep 5
if $DOCKER_COMPOSE_CMD ps app | grep -q "Up"; then
    echo -e "${GREEN}✅ Приложение успешно запущено!${NC}"
    echo ""
    echo -e "${GREEN}📊 Статус сервисов:${NC}"
    $DOCKER_COMPOSE_CMD ps
    echo ""
    echo -e "${GREEN}📝 Полезные команды:${NC}"
    echo -e "  ${YELLOW}make logs${NC}     - Показать логи приложения"
    echo -e "  ${YELLOW}make logs-db${NC}  - Показать логи базы данных"
    echo -e "  ${YELLOW}make stop${NC}     - Остановить приложение"
    echo -e "  ${YELLOW}make status${NC}   - Показать статус сервисов"
    echo ""
    echo -e "${GREEN}🎉 Бот готов к работе!${NC}"
else
    echo -e "${RED}❌ Ошибка запуска приложения!${NC}"
    echo -e "${YELLOW}Логи приложения:${NC}"
    $DOCKER_COMPOSE_CMD logs app
    exit 1
fi 