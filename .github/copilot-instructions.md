---
description: Coding guidelines and future feature considerations for the project.
# Future Features (not required initially)

Potential extensions:

- Redis temporary booking holds
- WebSocket real-time availability updates
- Google Calendar integration
- Email notifications
- Prometheus metrics
- Grafana dashboards
- OpenTelemetry tracing

Do not implement these unless the core booking workflow is complete.

---

# Coding Guidelines

When generating code:

1. Prefer simple maintainable solutions.
2. Avoid premature abstraction.
3. Follow Go idioms.
4. Use context.Context everywhere for database operations.
5. Handle errors explicitly.
6. Write unit tests for business logic.
7. Keep database logic inside repositories.
8. Keep handlers thin.
9. Use meaningful names.
10. Optimize for correctness over cleverness.

Before implementing new features:
- Explain the design.
- Confirm database impact.
- Then generate code.