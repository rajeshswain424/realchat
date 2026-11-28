# llm-client Specification

## Purpose

Defines how the agent talks to a language model over an OpenAI-compatible chat-completions API, including tool definitions, configuration and error handling.

## Requirements

### Requirement: Chat completion with conversation history
The system SHALL send an ordered list of messages to the configured model endpoint and return the model's reply message.

#### Scenario: Plain reply
- **WHEN** a conversation with a user message is sent and the model answers with text
- **THEN** the system returns an assistant message containing that text

#### Scenario: Multi-turn history is preserved
- **WHEN** a conversation with earlier user, assistant and tool messages is sent
- **THEN** the messages reach the model in the same order and with the same roles

### Requirement: Tool definitions and tool calls
The system SHALL send tool definitions with a request and SHALL return any tool calls the model requests, each with an id, a tool name and its raw JSON arguments.

#### Scenario: Model requests a tool
- **WHEN** the model replies with one or more tool calls
- **THEN** the returned assistant message contains every tool call with its id, name and arguments

#### Scenario: Tool result is sent back
- **WHEN** a tool message that references a tool call id is included in the next request
- **THEN** the system sends it with the `tool` role and that same id

### Requirement: Configuration from environment
The system SHALL read the API key, base URL and model name from environment variables and SHALL NOT require them to be stored in the repository.

#### Scenario: Missing API key
- **WHEN** the API key variable is not set
- **THEN** the system fails before any network request with a message naming the missing variable

#### Scenario: Custom base URL
- **WHEN** a base URL is configured
- **THEN** requests go to that base URL instead of the default endpoint

### Requirement: Error reporting
The system SHALL report non-success HTTP responses and network failures as errors that include the HTTP status and the provider's error message when available, and SHALL NOT include the API key.

#### Scenario: Provider rejects the request
- **WHEN** the endpoint returns a non-2xx status with an error body
- **THEN** the system returns an error containing the status code and the provider's message

#### Scenario: Request times out
- **WHEN** the endpoint does not respond within the configured timeout
- **THEN** the system returns a timeout error instead of waiting indefinitely
