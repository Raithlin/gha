package output

import (
	"fmt"
	"io"

	"github.com/raithlin/gha/internal/release"
)

// ReleaseAssets renders the same bounded inventory in text or structured form.
func ReleaseAssets(writer io.Writer, format Format, result *release.AssetInventory) error {
	if format != Text {
		return structured(writer, format, result)
	}
	styles := newStyles(writer)
	completeness := "complete"
	if !result.Complete {
		completeness = "truncated"
	}
	if _, err := fmt.Fprintf(writer, "%s\n%s: %s\n%s: %s\n%s: %d\n%s: %t\n%s: %t\n%s: %s (%d/%d)\n",
		styles.heading("Release assets"), styles.label("Repository"), sanitizeTerminal(result.Repository.String()),
		styles.label("Tag"), sanitizeTerminal(result.Tag), styles.label("Release ID"), result.ReleaseID,
		styles.label("Draft"), result.Draft, styles.label("Prerelease"), result.Prerelease,
		styles.label("Inventory"), completeness, len(result.Assets), result.Limit); err != nil {
		return err
	}
	for _, asset := range result.Assets {
		if _, err := fmt.Fprintf(writer, "  %d  %s  %d bytes  %s\n", asset.ID, sanitizeTerminal(asset.Name), asset.Size, sanitizeTerminal(asset.Digest)); err != nil {
			return err
		}
	}
	return nil
}

// ReleaseVerification renders per-asset evidence and an overall state.
func ReleaseVerification(writer io.Writer, format Format, result *release.Verification) error {
	if format != Text {
		return structured(writer, format, result)
	}
	styles := newStyles(writer)
	if _, err := fmt.Fprintf(writer, "%s\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n",
		styles.heading("Release asset verification"), styles.label("Repository"), sanitizeTerminal(result.Repository.String()),
		styles.label("Tag"), sanitizeTerminal(result.Tag), styles.label("State"), sanitizeTerminal(result.State),
		styles.label("Output directory"), sanitizeTerminal(result.OutputDir), styles.label("Manifest"), sanitizeTerminal(result.ManifestState)); err != nil {
		return err
	}
	for _, asset := range result.Assets {
		if _, err := fmt.Fprintf(writer, "  %s: provider %s, manifest %s, %s\n", sanitizeTerminal(asset.Name),
			sanitizeTerminal(asset.ProviderDigestState), sanitizeTerminal(asset.ManifestState), sanitizeTerminal(asset.Destination)); err != nil {
			return err
		}
		if asset.Message != "" {
			if _, err := fmt.Fprintf(writer, "    %s\n", sanitizeTerminal(asset.Message)); err != nil {
				return err
			}
		}
	}
	return nil
}
