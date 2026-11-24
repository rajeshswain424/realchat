# Spec Delta

## Purpose

Defines the read-only tools that let the agent browse and search a folder of markdown notes without being able to read or change anything outside it.

## ADDED Requirements

### Requirement: List files
The `list_files` tool SHALL return the relative paths of the markdown files under the notes folder, including subfolders, in a stable sorted order.

#### Scenario: Notes with subfolders
- **WHEN** the notes folder contains `a.md` and `work/b.md`
- **THEN** the tool returns `a.md` and `work/b.md`

#### Scenario: Empty folder
- **WHEN** the notes folder has no markdown files
- **THEN** the tool returns a result stating that no files were found

### Requirement: Read file
The `read_file` tool SHALL return the text of one markdown file identified by a path relative to the notes folder.

#### Scenario: Existing file
- **WHEN** the path names an existing note
- **THEN** the tool returns its text

#### Scenario: Missing file
- **WHEN** the path does not exist
- **THEN** the tool returns an error saying the file was not found

### Requirement: Search text
The `search_text` tool SHALL return the matching lines for a query across all notes, each with its file path and line number, matching case-insensitively.

#### Scenario: Query matches
- **WHEN** the query appears in two notes
- **THEN** the tool returns each matching line with its path and line number

#### Scenario: No matches
- **WHEN** the query appears nowhere
- **THEN** the tool returns a result stating that nothing matched

### Requirement: Confinement to the notes folder
The tools SHALL NOT read any file outside the notes folder, including through `..` segments, absolute paths or symbolic links.

#### Scenario: Parent directory traversal
- **WHEN** `read_file` is given `../secret.txt`
- **THEN** the tool returns an error and reads nothing

#### Scenario: Absolute path
- **WHEN** `read_file` is given an absolute path outside the notes folder
- **THEN** the tool returns an error and reads nothing

#### Scenario: Symlink pointing outside
- **WHEN** a note inside the folder is a symbolic link to a file outside it
- **THEN** `read_file` returns an error and `search_text` does not search that file

### Requirement: Bounded output
Each tool SHALL limit the size of its result and SHALL state in the result when output was truncated.

#### Scenario: Large file
- **WHEN** `read_file` targets a file larger than the output limit
- **THEN** the tool returns the first part of the file followed by a note that it was truncated

#### Scenario: Many matches
- **WHEN** `search_text` finds more matches than the limit
- **THEN** the tool returns up to the limit and states that more matches exist

### Requirement: Tool schemas
The system SHALL expose each tool to the model with a name, a description and a JSON schema for its arguments.

#### Scenario: Definitions sent to the model
- **WHEN** a model request is built for the agent
- **THEN** it includes definitions for `list_files`, `read_file` and `search_text`
