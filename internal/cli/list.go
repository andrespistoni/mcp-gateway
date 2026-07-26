package cli

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"mcp-gateway/internal/app"
)

type Application interface {
	List(context.Context) ([]app.ListItem, error)
}

func writeList(ctx context.Context, output io.Writer, application Application) error {
	items, err := application.List(ctx)
	if err != nil {
		return err
	}
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "NAME\tSTATUS\tPREFIX\tBINARY"); err != nil {
		return err
	}
	for _, item := range items {
		binary := item.Binary
		if item.Fallback {
			binary += " (fallback PATH)"
		}
		if item.Diagnostic != "" {
			binary += " (" + item.Diagnostic + ")"
		}
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", item.Name, item.Status, item.Prefix, binary); err != nil {
			return err
		}
	}
	return writer.Flush()
}
