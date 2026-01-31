# Resources Folder

This folder contains additional resources for the OtisBrain system:

## Subfolders

### mcpserver/
Configuration files for the Model Context Protocol server that enables communication between the AI system and external tools.

### skills/
YAML definitions and associated code for skills that extend the AI assistant's capabilities. Skills allow the AI to perform specific tasks like retrieving cluster information, analyzing events, and providing recommendations.

## Skills System

The skills system allows the AI assistant to perform various tasks by calling specialized functions. Each skill is defined in a YAML file that specifies:

- Input parameters
- Output format
- Execution method
- Metadata

Skills are implemented as Go functions in the `ai/tools` package and registered with the AI system to enable natural language interaction.