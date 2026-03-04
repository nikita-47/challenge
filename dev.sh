#!/bin/bash
# Dev server management for AI Challenge project

GO_PORT=8080
VITE_PORTS="5173 5174"
LMS="$HOME/.lmstudio/bin/lms"
LMS_MODEL="qwen2.5-0.5b-instruct-mlx"
LMS_PORT=1234

stop_go() {
    local pids
    pids=$(lsof -i :$GO_PORT -t 2>/dev/null)
    if [ -n "$pids" ]; then
        echo "$pids" | xargs kill 2>/dev/null
        echo "Go server stopped (port $GO_PORT)"
    else
        echo "Go server not running"
    fi
}

start_go() {
    local pids
    pids=$(lsof -i :$GO_PORT -t 2>/dev/null)
    if [ -n "$pids" ]; then
        echo "Go server already running on port $GO_PORT (PID: $pids)"
        return
    fi
    go run ./backend --server --port $GO_PORT &
    echo "Go server starting on port $GO_PORT"
}

stop_vite() {
    local found=false
    for port in $VITE_PORTS; do
        local pids
        pids=$(lsof -i :$port -t 2>/dev/null)
        if [ -n "$pids" ]; then
            echo "$pids" | xargs kill 2>/dev/null
            echo "Vite stopped (port $port)"
            found=true
        fi
    done
    if [ "$found" = false ]; then
        echo "Vite not running"
    fi
}

start_vite() {
    for port in $VITE_PORTS; do
        local pids
        pids=$(lsof -i :$port -t 2>/dev/null)
        if [ -n "$pids" ]; then
            echo "Vite already running on port $port (PID: $pids)"
            return
        fi
    done
    cd frontend && npm run dev &
    echo "Vite starting"
}

stop_lms() {
    if "$LMS" server status 2>&1 | grep -q "is running"; then
        "$LMS" server stop
        echo "LM Studio server stopped"
    else
        echo "LM Studio server not running"
    fi
}

start_lms() {
    if "$LMS" server status 2>&1 | grep -q "is running"; then
        echo "LM Studio server already running"
    else
        "$LMS" server start --port "$LMS_PORT"
        echo "LM Studio server starting on port $LMS_PORT"
    fi
    "$LMS" load "$LMS_MODEL" --yes
    echo "Model $LMS_MODEL loaded"
}

start_go_test() {
    local pids
    pids=$(lsof -i :$GO_PORT -t 2>/dev/null)
    if [ -n "$pids" ]; then
        echo "Go server already running on port $GO_PORT (PID: $pids)"
        return
    fi
    go run ./backend --server --port $GO_PORT --base-url "http://localhost:$LMS_PORT" --model "$LMS_MODEL" &
    echo "Go server starting on port $GO_PORT (base-url: localhost:$LMS_PORT, model: $LMS_MODEL)"
}

stop_playwright() {
    local pids
    pids=$(pgrep -f "@playwright/mcp" 2>/dev/null)
    if [ -n "$pids" ]; then
        echo "$pids" | xargs kill 2>/dev/null
        echo "Playwright MCP stopped"
    else
        echo "Playwright MCP not running"
    fi
}

status() {
    echo "=== Dev Server Status ==="
    local go_pids
    go_pids=$(lsof -i :$GO_PORT -t 2>/dev/null)
    if [ -n "$go_pids" ]; then
        echo "Go server:      running (port $GO_PORT, PID: $go_pids)"
    else
        echo "Go server:      not running"
    fi

    local vite_running=false
    for port in $VITE_PORTS; do
        local vite_pids
        vite_pids=$(lsof -i :$port -t 2>/dev/null)
        if [ -n "$vite_pids" ]; then
            echo "Vite:           running (port $port, PID: $vite_pids)"
            vite_running=true
        fi
    done
    if [ "$vite_running" = false ]; then
        echo "Vite:           not running"
    fi

    if "$LMS" server status 2>&1 | grep -q "is running"; then
        echo "LM Studio:      running (port $LMS_PORT)"
    else
        echo "LM Studio:      not running"
    fi

    local pw_pids
    pw_pids=$(pgrep -f "@playwright/mcp" 2>/dev/null)
    if [ -n "$pw_pids" ]; then
        echo "Playwright MCP: running (PID: $pw_pids)"
    else
        echo "Playwright MCP: not running"
    fi
}

case "${1:-}" in
    start-go)     start_go ;;
    stop-go)      stop_go ;;
    restart-go)   stop_go && sleep 1 && start_go ;;
    start-vite)   start_vite ;;
    stop-vite)    stop_vite ;;
    stop-playwright) stop_playwright ;;
    start)        start_go; start_vite ;;
    start-lms)    start_lms ;;
    stop-lms)     stop_lms ;;
    start-test)   start_lms; start_go_test; start_vite ;;
    stop-test)    stop_go; stop_vite; stop_lms ;;
    stop)         stop_go; stop_vite; stop_playwright ;;
    status)       status ;;
    *)
        echo "Usage: ./dev.sh {start|stop|status|start-go|stop-go|restart-go|start-vite|stop-vite|stop-playwright|start-lms|stop-lms|start-test|stop-test}"
        exit 1
        ;;
esac
