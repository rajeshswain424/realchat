# agent-eval Specification

## Purpose

Defines a repeatable evaluation that runs fixed questions through the agent against a sample notes folder and reports how many it answers correctly, giving a baseline to compare later changes against.

## Requirements

### Requirement: Fixed eval set
The system SHALL keep a versioned eval set in the repository, where each case has a question, a check for the expected answer, and optionally the tool the agent is expected to use.

#### Scenario: Eval set loads
- **WHEN** the eval runner starts
- **THEN** it loads every case from the eval set and fails with a clear message if a case is malformed

### Requirement: Sample notes corpus
The system SHALL include a sample notes folder whose contents the eval answers are derived from, so results do not depend on a user's private notes.

#### Scenario: Eval runs on a fresh checkout
- **WHEN** the eval runs on a fresh checkout with only an API key set
- **THEN** it uses the bundled sample notes

### Requirement: Case checking
The runner SHALL mark a case as passed only when the final answer satisfies its expected-answer check and, if a required tool is given, that tool was called.

#### Scenario: Correct answer
- **WHEN** the final answer contains the expected text
- **THEN** the case passes

#### Scenario: Wrong answer
- **WHEN** the final answer does not contain the expected text
- **THEN** the case fails and the report shows the question and the actual answer

#### Scenario: Required tool not used
- **WHEN** the answer is correct but the required tool was never called
- **THEN** the case fails with a reason naming the missing tool

### Requirement: Pass-rate report
The runner SHALL print per-case results and a summary with passed count, total count and pass rate, and SHALL exit with a non-zero status when any case fails.

#### Scenario: All pass
- **WHEN** every case passes
- **THEN** the summary shows a 100% pass rate and the exit status is 0

#### Scenario: Some fail
- **WHEN** at least one case fails
- **THEN** the summary shows the lower rate and the exit status is non-zero

### Requirement: Errors count as failures
The runner SHALL treat an agent error, such as a network failure or the iteration cap, as a failed case and SHALL continue with the remaining cases.

#### Scenario: One case errors
- **WHEN** the agent returns an error for one case
- **THEN** that case is reported as failed with the error, and later cases still run
