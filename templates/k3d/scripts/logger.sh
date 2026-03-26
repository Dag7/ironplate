#!/bin/bash
# =============================================================================
# Simple logging utilities for cluster management scripts
# =============================================================================
Green='\033[0;32m'
Yellow='\033[0;33m'
Red='\033[0;31m'
Blue='\033[1;34m'
Magenta='\033[0;35m'
NC='\033[0m'

logger_highlight() {
    echo -e "${Magenta}${1}${NC}"
}

logger_info() {
    echo -e "${Blue}${1}${NC}"
}

logger_success() {
    echo -e "${Green}${1}${NC}"
}

logger_warning() {
    echo -e "${Yellow}${1}${NC}"
}

logger_error() {
    echo -e "${Red}${1}${NC}"
}
