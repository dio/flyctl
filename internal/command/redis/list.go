package redis

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	fly "github.com/superfly/fly-go"
	"github.com/superfly/flyctl/gql"
	"github.com/superfly/flyctl/iostreams"

	"github.com/superfly/flyctl/internal/command"
	"github.com/superfly/flyctl/internal/flag"
	"github.com/superfly/flyctl/internal/render"
)

func newList() (cmd *cobra.Command) {
	const (
		long  = `List Upstash Redis databases for an organization`
		short = long
		usage = "list"
	)

	cmd = command.New(usage, short, long, runList, command.RequireSession)

	cmd.Aliases = []string{"ls"}

	flag.Add(cmd,
		flag.Org(),
	)

	return cmd
}

func runList(ctx context.Context) (err error) {
	var (
		out    = iostreams.FromContext(ctx).Out
		client = fly.ClientFromContext(ctx).GenqClient
	)

	response, err := gql.ListAddOns(ctx, client, "redis")

	var rows [][]string

	for _, addon := range response.AddOns.Nodes {
		options, _ := addon.Options.(map[string]interface{})
		var eviction = "Disabled"

		if options["eviction"] != nil && options["eviction"].(bool) {
			eviction = "Enabled"
		}

		rows = append(rows, []string{
			addon.Name,
			addon.Organization.Slug,
			addon.AddOnPlan.DisplayName,
			eviction,
			addon.PrimaryRegion,
			strings.Join(addon.ReadRegions, ","),
		})
	}

	_ = render.Table(out, "", rows, "Name", "Org", "Plan", "Eviction", "Primary Region", "Read Regions")

	return
}
