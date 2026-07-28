package cli

import (
	"context"
	"fmt"
	"io"

	"mcp-gateway/internal/app"
	"mcp-gateway/internal/diagnostics"
)

type doctorApplication interface {
	Doctor(context.Context, app.DoctorRequest) (app.DoctorReport, error)
}

func writeDoctor(ctx context.Context, output io.Writer, application doctorApplication, command parsedCommand) error {
	report, err := application.Doctor(ctx, app.DoctorRequest{Verbose: command.verbose})
	if err != nil {
		return err
	}
	for _, check := range report.Checks {
		state := "OK"
		if !check.OK() {
			state = "ERROR"
		}
		if command.verbose && check.Detail != "" {
			if _, err := fmt.Fprintf(output, "%s\t%s\t%s\n", check.Name, state, diagnostics.RedactText(check.Detail)); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(output, "%s\t%s\n", check.Name, state); err != nil {
			return err
		}
	}
	if !report.OK() {
		return diagnostics.NewFault(diagnostics.Unavailable, "doctor detectó comprobaciones fallidas", nil)
	}
	return nil
}
