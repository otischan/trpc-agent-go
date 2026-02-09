#!/bin/bash

# Script to start all MCP servers based on configuration
# This script reads run-mcp-config.yaml and starts all enabled MCP servers

set -e  # Exit on any error

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/run-mcp-config.yaml"
PID_DIR="$SCRIPT_DIR/pids"
LOG_DIR="$SCRIPT_DIR/logs"

# Create necessary directories
mkdir -p "$PID_DIR" "$LOG_DIR"

# Function to start an MCP server
start_server() {
    local name=$1
    local enabled=$2
    local port=$3
    local binary_path=$4
    local args=$5
    local server_url=$6

    if [ "$enabled" = "true" ]; then
        echo "Starting $name on port $port..."

        # Check if binary exists
        if [ ! -f "$binary_path" ]; then
            echo "Error: Binary not found at $binary_path"
            return 1
        fi

        # Start the server in background and capture PID
        "$binary_path" $args > "$LOG_DIR/$name.log" 2>&1 &
        SERVER_PID=$!

        # Save PID to file
        echo $SERVER_PID > "$PID_DIR/$name.pid"

        # Wait a moment for server to start
        sleep 3

        # Check if server is running
        if kill -0 $SERVER_PID 2>/dev/null; then
            echo "$name started successfully with PID: $SERVER_PID, listening on $server_url"
        else
            echo "Failed to start $name. Check logs at $LOG_DIR/$name.log"
            return 1
        fi
    else
        echo "Skipping disabled server: $name"
    fi
}

# Function to stop all MCP servers
stop_all_servers() {
    echo "Stopping all MCP servers..."

    for pid_file in "$PID_DIR"/*.pid; do
        if [ -f "$pid_file" ]; then
            SERVER_PID=$(cat "$pid_file")
            SERVER_NAME=$(basename "$pid_file" .pid)

            if kill -0 $SERVER_PID 2>/dev/null; then
                echo "Stopping $SERVER_NAME (PID: $SERVER_PID)..."
                kill $SERVER_PID

                # Wait for graceful shutdown
                for i in {1..10}; do
                    if ! kill -0 $SERVER_PID 2>/dev/null; then
                        echo "$SERVER_NAME stopped successfully"
                        break
                    fi
                    sleep 1
                done

                # Force kill if still running
                if kill -0 $SERVER_PID 2>/dev/null; then
                    echo "Force killing $SERVER_NAME..."
                    kill -9 $SERVER_PID
                fi
            else
                echo "$SERVER_NAME (PID: $SERVER_PID) is not running"
            fi

            # Remove PID file
            rm -f "$pid_file"
        fi
    done
}

# Function to check server status
check_status() {
    for pid_file in "$PID_DIR"/*.pid; do
        if [ -f "$pid_file" ]; then
            SERVER_PID=$(cat "$pid_file")
            SERVER_NAME=$(basename "$pid_file" .pid)

            if kill -0 $SERVER_PID 2>/dev/null; then
                echo "$SERVER_NAME is running with PID: $SERVER_PID"
            else
                echo "$SERVER_NAME (PID: $SERVER_PID) is not running"
                rm -f "$pid_file"
            fi
        fi
    done
}

# Main execution
case "${1:-status}" in
    start)
        echo "Starting MCP servers based on configuration..."

        # Read the configuration file and start enabled servers
        # Using yq or jq if available, otherwise using awk/sed for parsing
        if command -v yq &> /dev/null; then
            # Using yq to parse YAML
            while IFS= read -r line; do
                if [[ $line =~ ^[[:space:]]*name:[[:space:]]+"([^"]+)" ]]; then
                    name="${BASH_REMATCH[1]}"
                elif [[ $line =~ ^[[:space:]]*enabled:[[:space:]]+(true|false) ]]; then
                    enabled="${BASH_REMATCH[1]}"
                elif [[ $line =~ ^[[:space:]]*binary_path:[[:space:]]+"([^"]+)" ]]; then
                    binary_path="${BASH_REMATCH[1]}"
                elif [[ $line =~ ^[[:space:]]*args:[[:space:]]* ]]; then
                    # Extract args array
                    args_line=$(echo "$line" | sed 's/args:[[:space:]]*//')
                    args=$(echo "$args_line" | sed 's/^\[\(.*\)\]$/\1/' | sed 's/"//g' | sed 's/,//g')
                fi
                
                # When we have all required fields and server is enabled, start it
                if [[ -n "$name" && -n "$enabled" && -n "$binary_path" ]]; then
                    if [[ "$enabled" == "true" ]]; then
                        echo "Processing server: $name"
                        # Convert args array to string format for execution
                        arg_string=""
                        if [[ -n "$args" ]]; then
                            for arg in $args; do
                                arg_string="$arg_string \"$arg\""
                            done
                        fi
                        
                        start_server "$name" "$enabled" "unknown" "$binary_path" "$arg_string" "unknown"
                    else
                        echo "Skipping disabled server: $name"
                    fi
                    
                    # Reset variables for next iteration
                    name=""
                    enabled=""
                    binary_path=""
                    args=""
                fi
            done < "$CONFIG_FILE"
        else
            # Fallback to grep/sed approach for parsing YAML
            # Extract server configurations
            declare -a server_names
            declare -a server_enabled
            declare -a server_binary_paths
            declare -a server_args
            
            # Read the config file and extract server information
            server_count=0
            current_server=""
            
            while IFS= read -r line; do
                # Check if this is a new server entry
                if [[ $line =~ ^'[[:space:]]*-[[:space:]]*name:' ]] || [[ $line =~ ^'-[[:space:]]*name:' ]]; then
                    # Extract server name
                    name=$(echo "$line" | sed -E 's/^[[:space:]]*-[[:space:]]*name:[[:space:]]*"([^"]+)".*/\1/')
                    current_server="$name"
                    
                    # Get the enabled status for this server
                    enabled=$(grep -A 10 "name: \"$name\"" "$CONFIG_FILE" | grep "enabled:" | head -1 | sed 's/enabled:[[:space:]]*//' | tr -d '"')
                    
                    # Get the binary path for this server
                    binary_path=$(grep -A 10 "name: \"$name\"" "$CONFIG_FILE" | grep "binary_path:" | head -1 | sed 's/binary_path:[[:space:]]*//' | tr -d '"')
                    
                    # Get the args for this server
                    args_block=$(grep -A 15 "name: \"$name\"" "$CONFIG_FILE" | sed -n '/args:/,/^[^-]/p' | head -n -1)
                    args=$(echo "$args_block" | grep -v "args:" | sed 's/[[:space:]]*- //' | sed 's/"//g' | tr '\n' ' ')
                    
                    # Start the server if enabled
                    if [[ "$enabled" == "true" ]]; then
                        echo "Starting server: $name"
                        
                        # Convert args to proper format
                        formatted_args=""
                        for arg in $args; do
                            if [[ -n "$arg" ]]; then
                                formatted_args="$formatted_args \"$arg\""
                            fi
                        done
                        
                        start_server "$name" "$enabled" "unknown" "$binary_path" "$formatted_args" "unknown"
                    else
                        echo "Skipping disabled server: $name"
                    fi
                fi
            done < "$CONFIG_FILE"
        fi

        echo "All enabled MCP servers started."
        ;;
    stop)
        stop_all_servers
        ;;
    restart)
        stop_all_servers
        sleep 2
        $0 start
        ;;
    status)
        check_status
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        exit 1
        ;;
esac