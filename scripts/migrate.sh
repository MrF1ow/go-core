#!/bin/bash

# Database Migration Script
# Interactive tool for managing database migrations

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}Database Migration Tool${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""
echo "1. Apply pending migrations"
echo "2. Rollback last migration"
echo "3. List available migrations"
echo "4. Show migration status"
echo "5. Backup database"
echo "6. Test database connection"
echo "0. Exit"
echo ""
echo -e "${YELLOW}Enter your choice (0-6): ${NC}"
read -r choice

case $choice in
    1)
        "$SCRIPT_DIR/apply_pending_migrations.sh"
    ;;

    2)
        "$SCRIPT_DIR/rollback_last_migration.sh"
    ;;

    3)
        echo -e "${GREEN}Available Migrations:${NC}"
        echo ""
        if [ -d "$PROJECT_ROOT/migrations" ]; then
            ls -1 "$PROJECT_ROOT/migrations/"*.sql 2>/dev/null | grep -v '_rollback\.sql$' || echo "No migration files found"
        else
            echo -e "${RED}Migrations directory not found!${NC}"
        fi
    ;;

    4)
        echo -e "${GREEN}Tracked Migrations:${NC}"
        echo ""
        docker exec -i auth_db psql -U postgres -d auth_db -c \
            "SELECT version, applied_at FROM schema_migrations ORDER BY applied_at;" 2>/dev/null \
            || echo -e "${RED}Could not query migration status. Is the database running?${NC}"
    ;;

    5)
        "$SCRIPT_DIR/backup_db.sh"
    ;;

    6)
        echo -e "${GREEN}Testing Database Connection${NC}"
        echo ""
        if docker exec auth_db psql -U postgres -d auth_db -c "SELECT version();" > /dev/null 2>&1; then
            echo -e "${GREEN}Connection successful!${NC}"
            docker exec auth_db psql -U postgres -d auth_db -c "SELECT version();"
        else
            echo -e "${RED}Connection failed!${NC}"
            echo "Start the database with: make docker-dev"
        fi
    ;;

    0)
        echo "Exiting..."
        exit 0
    ;;

    *)
        echo -e "${RED}Invalid choice.${NC}"
        exit 1
    ;;
esac
