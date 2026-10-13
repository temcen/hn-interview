# Proyecto: interview

## Comandos (completar al elegir el stack)
- Instalar: nada (Go 1.24+, solo stdlib)
- Test: `go test -race ./...`
- Lint/format: `go vet ./... && gofmt -l .`
- Build/run: `go run ./cmd/hncapture` (ver `README.md` para subcomandos)

## Forma de trabajo
- Antes de implementar: aclarar requisitos y proponer un diseño corto.
- TDD: test que falla primero, luego el código mínimo, luego refactor.
- Cambios pequeños y enfocados; un commit por cambio lógico con mensaje claro.
- No inventar APIs ni librerías: verificar en el código o con Context7 (MCP).
- Antes de dar algo por terminado: correr tests y lint y reportar el resultado real.
- Leer el código existente y seguir sus convenciones antes de añadir patrones nuevos.

## Seguridad
- Nunca leer ni escribir `.env` ni archivos de credenciales (hook lo bloquea).
- No usar `git push --force`, `rm -rf` ni `curl | sh`.

## Herramientas del proyecto
- Subagentes: `code-reviewer`, `test-writer`, `debugger` (`.claude/agents/`).
- Comandos: `/review`, `/test-plan`, `/explain-arch` (`.claude/commands/`).
- MCP: Context7 (docs de librerías), Playwright (verificación en navegador).
