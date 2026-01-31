#!/bin/bash

# Script for safe file operations
OPERATION=${1:-"read"}
SOURCE_PATH="$2"
DESTINATION_PATH="$3"
CONTENT="$4"
FORCE="${5:-false}"

# Function to validate file path
validate_path() {
    local path="$1"
    
    # Check if path contains dangerous patterns
    if [[ "$path" =~ \.\./ ]] || [[ "$path" == *"../"* ]] || [[ "$path" == "/" ]]; then
        echo "ERROR: Invalid path detected - potential path traversal attempt: $path" >&2
        return 1
    fi
    
    # Check if path is empty
    if [ -z "$path" ]; then
        echo "ERROR: Path cannot be empty" >&2
        return 1
    fi
    
    return 0
}

# Function to check if file is safe to operate on
is_safe_to_operate() {
    local path="$1"
    local operation="$2"
    
    # Check if path exists and is a regular file/directory
    if [ "$operation" != "create_dir" ] && [ "$operation" != "symlink" ] && [ "$operation" != "write" ]; then
        if [ ! -e "$path" ]; then
            echo "ERROR: Path does not exist: $path" >&2
            return 1
        fi
    fi
    
    # Additional safety checks could be added here
    return 0
}

# Main operation switch
case "$OPERATION" in
    "read")
        if ! validate_path "$SOURCE_PATH"; then
            exit 1
        fi
        
        if ! is_safe_to_operate "$SOURCE_PATH" "read"; then
            exit 1
        fi
        
        if [ ! -f "$SOURCE_PATH" ]; then
            echo "ERROR: Cannot read - not a file: $SOURCE_PATH" >&2
            exit 1
        fi
        
        if [ ! -r "$SOURCE_PATH" ]; then
            echo "ERROR: Cannot read - no read permission: $SOURCE_PATH" >&2
            exit 1
        fi
        
        cat "$SOURCE_PATH"
        ;;
        
    "write")
        if ! validate_path "$SOURCE_PATH"; then
            exit 1
        fi
        
        # Check if parent directory exists and is writable
        local dir_path=$(dirname "$SOURCE_PATH")
        if [ ! -d "$dir_path" ]; then
            echo "ERROR: Parent directory does not exist: $dir_path" >&2
            exit 1
        fi
        
        if [ ! -w "$dir_path" ]; then
            echo "ERROR: No write permission to parent directory: $dir_path" >&2
            exit 1
        fi
        
        # If file exists and force is not true, check if we should proceed
        if [ -e "$SOURCE_PATH" ] && [ "$FORCE" != "true" ]; then
            echo "ERROR: File already exists: $SOURCE_PATH - use force=true to overwrite" >&2
            exit 1
        fi
        
        # Write content to file
        echo "$CONTENT" > "$SOURCE_PATH"
        echo "Successfully wrote to $SOURCE_PATH"
        ;;
        
    "copy")
        if ! validate_path "$SOURCE_PATH"; then
            exit 1
        fi
        
        if ! validate_path "$DESTINATION_PATH"; then
            exit 1
        fi
        
        if ! is_safe_to_operate "$SOURCE_PATH" "copy"; then
            exit 1
        fi
        
        if [ ! -e "$SOURCE_PATH" ]; then
            echo "ERROR: Source path does not exist: $SOURCE_PATH" >&2
            exit 1
        fi
        
        if [ -e "$DESTINATION_PATH" ] && [ "$FORCE" != "true" ]; then
            echo "ERROR: Destination already exists: $DESTINATION_PATH - use force=true to overwrite" >&2
            exit 1
        fi
        
        # Copy file
        cp "$SOURCE_PATH" "$DESTINATION_PATH"
        echo "Successfully copied $SOURCE_PATH to $DESTINATION_PATH"
        ;;
        
    "move")
        if ! validate_path "$SOURCE_PATH"; then
            exit 1
        fi
        
        if ! validate_path "$DESTINATION_PATH"; then
            exit 1
        fi
        
        if ! is_safe_to_operate "$SOURCE_PATH" "move"; then
            exit 1
        fi
        
        if [ ! -e "$SOURCE_PATH" ]; then
            echo "ERROR: Source path does not exist: $SOURCE_PATH" >&2
            exit 1
        fi
        
        if [ -e "$DESTINATION_PATH" ] && [ "$FORCE" != "true" ]; then
            echo "ERROR: Destination already exists: $DESTINATION_PATH - use force=true to overwrite" >&2
            exit 1
        fi
        
        # Move file
        mv "$SOURCE_PATH" "$DESTINATION_PATH"
        echo "Successfully moved $SOURCE_PATH to $DESTINATION_PATH"
        ;;
        
    "delete")
        if ! validate_path "$SOURCE_PATH"; then
            exit 1
        fi
        
        if ! is_safe_to_operate "$SOURCE_PATH" "delete"; then
            exit 1
        fi
        
        if [ ! -e "$SOURCE_PATH" ]; then
            echo "ERROR: Path does not exist: $SOURCE_PATH" >&2
            exit 1
        fi
        
        # Confirm deletion for safety
        rm "$SOURCE_PATH"
        echo "Successfully deleted $SOURCE_PATH"
        ;;
        
    "list")
        if ! validate_path "$SOURCE_PATH"; then
            exit 1
        fi
        
        if ! is_safe_to_operate "$SOURCE_PATH" "list"; then
            exit 1
        fi
        
        if [ ! -d "$SOURCE_PATH" ]; then
            echo "ERROR: Cannot list - not a directory: $SOURCE_PATH" >&2
            exit 1
        fi
        
        if [ ! -r "$SOURCE_PATH" ]; then
            echo "ERROR: Cannot list - no read permission: $SOURCE_PATH" >&2
            exit 1
        fi
        
        # List directory contents
        ls -la "$SOURCE_PATH"
        ;;
        
    "create_dir")
        if ! validate_path "$SOURCE_PATH"; then
            exit 1
        fi
        
        # Create directory with parents as needed
        mkdir -p "$SOURCE_PATH"
        echo "Successfully created directory $SOURCE_PATH"
        ;;
        
    "chmod")
        if ! validate_path "$SOURCE_PATH"; then
            exit 1
        fi
        
        if ! is_safe_to_operate "$SOURCE_PATH" "chmod"; then
            exit 1
        fi
        
        if [ ! -e "$SOURCE_PATH" ]; then
            echo "ERROR: Path does not exist: $SOURCE_PATH" >&2
            exit 1
        fi
        
        # Change file permissions
        chmod "$CONTENT" "$SOURCE_PATH"
        echo "Successfully changed permissions of $SOURCE_PATH to $CONTENT"
        ;;
        
    "symlink")
        if ! validate_path "$SOURCE_PATH"; then
            exit 1
        fi
        
        if ! validate_path "$DESTINATION_PATH"; then
            exit 1
        fi
        
        if [ -e "$DESTINATION_PATH" ] && [ "$FORCE" != "true" ]; then
            echo "ERROR: Symbolic link destination already exists: $DESTINATION_PATH - use force=true to overwrite" >&2
            exit 1
        fi
        
        # Create symbolic link
        ln -s "$SOURCE_PATH" "$DESTINATION_PATH"
        echo "Successfully created symbolic link from $SOURCE_PATH to $DESTINATION_PATH"
        ;;
        
    *)
        echo "ERROR: Unsupported operation: $OPERATION" >&2
        echo "Supported operations: read, write, copy, move, delete, list, create_dir, chmod, symlink" >&2
        exit 1
        ;;
esac