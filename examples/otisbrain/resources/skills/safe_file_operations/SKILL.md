---
name: safe_file_operations
description: 安全地执行文件操作，包括读取、写入、复制、移动和删除文件
---

Overview

提供安全的文件操作功能，包括读取、写入、复制、移动和删除文件。该技能包含安全检查以防止意外覆盖或删除重要文件。

Parameters

- operation: 操作类型 (read, write, copy, move, delete, list, create_dir, chmod, chown, symlink, default: read)
- source_path: 源文件路径
- destination_path: 目标文件路径（可选，用于copy、move操作）
- content: 文件内容（仅用于write操作）
- recursive: 是否递归操作目录 (true, false, default: false)
- preserve_attributes: 是否保留文件属性 (true, false, default: false)
- force: 是否强制操作（覆盖现有文件） (true, false, default: false)

Examples

1) 读取文件内容

   Command:

   bash scripts/safe_file_operations.sh read "{{ inputs.source_path }}"

2) 写入内容到文件

   Command:

   bash scripts/safe_file_operations.sh write "{{ inputs.source_path }}" "{{ inputs.content }}"

3) 复制文件

   Command:

   bash scripts/safe_file_operations.sh copy "{{ inputs.source_path }}" "{{ inputs.destination_path }}" "{{ inputs.force }}"

4) 移动文件

   Command:

   bash scripts/safe_file_operations.sh move "{{ inputs.source_path }}" "{{ inputs.destination_path }}" "{{ inputs.force }}"

5) 删除文件

   Command:

   bash scripts/safe_file_operations.sh delete "{{ inputs.source_path }}"

6) 列出目录内容

   Command:

   bash scripts/safe_file_operations.sh list "{{ inputs.source_path }}" "{{ inputs.recursive }}"

7) 创建目录

   Command:

   bash scripts/safe_file_operations.sh create_dir "{{ inputs.source_path }}"

8) 修改文件权限

   Command:

   bash scripts/safe_file_operations.sh chmod "{{ inputs.source_path }}" "{{ inputs.content }}"

9) 创建符号链接

   Command:

   bash scripts/safe_file_operations.sh symlink "{{ inputs.source_path }}" "{{ inputs.destination_path }}"

Output Files

- None (outputs to stdout)