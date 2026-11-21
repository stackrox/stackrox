package main

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"text/tabwriter"
)

type tableRow struct {
	Image    image
	Tag      string
	Location imageLocation
}

func printTable(rows []tableRow, overlayTag string) {
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "NAME\tLOCAL\tREMOTE\tREPLACED")
	for _, row := range rows {
		local, remote, replaced := "", "", ""
		switch row.Location {
		case imageLocal:
			local = "✔"
		case imageRemote:
			remote = "✔"
		default:
		}
		// Collector/fact tags never match the main tag, so REPLACED would always be filled in.
		if row.Image.mainVersioned() && row.Tag != overlayTag {
			replaced = row.Tag
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", row.Image.name(), local, remote, replaced)
	}
	w.Flush()
}

// checkResolution fails if no image resolved to the overlay tag or if any image
// is unavailable (the deployment couldn't succeed), and warns about mixing overlay and base tags.
func checkResolution(res resolution, overlayTag string) error {
	anyOverlay := slices.ContainsFunc(res.rows, func(row tableRow) bool {
		return row.Tag == overlayTag
	})
	if !anyOverlay {
		return fmt.Errorf("no image found for overlay tag %s", overlayTag)
	}
	var missing []string
	for _, row := range res.rows {
		if row.Location == imageNotFound {
			missing = append(missing, row.Image.name())
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("images not found locally or remotely: %s", strings.Join(missing, ", "))
	}
	if len(res.onOverlay) > 0 && len(res.onBase) > 0 {
		log.Printf("⚠  tag mixing: overlay=[%s] base=[%s]\n",
			strings.Join(res.onOverlay, ", "), strings.Join(res.onBase, ", "))
	}
	return nil
}
