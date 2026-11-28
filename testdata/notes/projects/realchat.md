# RealChat

RealChat is a notes agent written in Go. It answers questions about a folder of markdown files.

## Stack
- Language: Go 1.22
- HTTP router: chi (planned for the API phase)
- Storage for traces: SQLite
- Model provider: Groq, using the model openai/gpt-oss-120b

## Milestones
- M3 is the agent loop. The iteration cap is 10 model calls.
- M5 is the first eval run. The target pass rate is 80 percent.

## Owner
Rajesh owns the project and works on it on weekends.
