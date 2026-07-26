package cli

import (
	"fmt"
	"io"
)

const helpText = `Uso: mcp-gateway <comando> [opciones]

Comandos:
  setup [--port N]
  discover [--write]
  add <name> --prefix <prefix__> --binary <ruta-o-nombre> [opciones]
  remove <name>
  enable <name>
  disable <name>
  list
  doctor [--verbose]
  serve [--port N]
  enable-daemon [--port N]
  disable-daemon
  restart
  register-project [--project-dir RUTA] [--port N]
  install-claude [--port N]
  version
  help
`

func writeHelp(output io.Writer) error {
	_, err := fmt.Fprint(output, helpText)
	return err
}
