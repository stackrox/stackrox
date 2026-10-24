package reportgenerator

import (
	"fmt"
	"html"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// This file builds the HTML body of report notification emails. The markup is
// intentionally email-safe: table-based layout with inline styles only (no
// external CSS, flexbox or grid), a fixed 600px centered container, web-safe
// fonts and no background images. Colors are resolved from the product's
// PatternFly v6 severity tokens (see severityColorByValue) so the email matches
// the in-product look and feel.
//
// The body references the branding logo via <img src="cid:logo.png">; the inline
// logo attachment itself is added by the email notifier (see
// central/notifiers/email Message.writeContentBytes).

const (
	colorInk    = "#151515"
	colorMuted  = "#6a6e73"
	colorBorder = "#e0e0e0"
	colorRowAlt = "#fafafa"
	fontStack   = "Arial,Helvetica,sans-serif"
)

// severityColorByValue maps a vulnerability severity to the hex color of the
// product's PatternFly v6 severity token. Kept in sync with the UI
// (ui/apps/platform/src/constants/severityColors.ts). Because email clients
// cannot resolve CSS custom properties, the token values are inlined as hex.
var severityColorByValue = map[storage.VulnerabilitySeverity]string{
	storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:  "#b1380b",
	storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY: "#ca6c0f",
	storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:  "#dca614",
	storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:       "#707070",
	storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY:   "#c7c7c7",
}

// statCard is a single highlighted count (e.g. deployed-image CVEs).
type statCard struct {
	value string
	label string
}

// detailRow is one row of the "Report configuration" table.
type detailRow struct {
	label string
	value string // plain text; HTML-escaped by the renderer
	mono  bool   // render the value in a monospace chip (used for raw filter queries)
	bold  bool   // render the value in bold (used for the config name)
}

// emailLayout captures everything needed to render a report email body. The
// same layout is used for workload and node reports; node reports simply omit
// the second stat card and the severity section.
type emailLayout struct {
	title      string // e.g. "Workload CVE Report"
	subtitle   string // plain text; HTML-escaped by the renderer
	introHTML  string // trusted HTML/text from the body template or a custom body
	statCards  []statCard
	noVulnsMsg string // when set, replaces the stat cards with a success banner
	// severities, when non-empty, renders the severity legend (colored dot + label).
	severities []storage.VulnerabilitySeverity
	detailRows []detailRow
	// hasAttachment controls the attachment note; false renders a "no file
	// attached" note instead.
	hasAttachment bool
	reportURL     string // when set, renders the "View report in console" button
}

func (l emailLayout) render() string {
	var b strings.Builder

	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f4f4f4;"><tr><td align="center" style="padding:24px 12px;">`)
	b.WriteString(fmt.Sprintf(`<table role="presentation" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px; max-width:600px; background-color:#ffffff; border:1px solid %s; border-radius:6px; overflow:hidden;">`, colorBorder))

	l.writeHeader(&b)
	l.writeIntro(&b)
	l.writeSummary(&b)
	l.writeSeverities(&b)
	l.writeDetails(&b)
	l.writeAttachmentNote(&b)
	l.writeButton(&b)
	l.writeFooter(&b)

	b.WriteString(`</table>`)
	b.WriteString(`</td></tr></table>`)
	return b.String()
}

func (l emailLayout) writeHeader(b *strings.Builder) {
	// White logo band, so the branding logo stays legible regardless of its
	// own colors, followed by a dark title bar.
	b.WriteString(fmt.Sprintf(`<tr><td style="padding:20px 32px; border-bottom:1px solid %s;">`, colorBorder))
	b.WriteString(`<img src="cid:logo.png" width="160" style="width:160px; height:auto; display:block;" alt="Red Hat Advanced Cluster Security">`)
	b.WriteString(`</td></tr>`)

	b.WriteString(fmt.Sprintf(`<tr><td style="background-color:%s; padding:22px 32px;">`, colorInk))
	b.WriteString(fmt.Sprintf(`<div style="font-family:%s; color:#ffffff; font-size:22px; font-weight:bold; line-height:1.3;">%s</div>`, fontStack, html.EscapeString(l.title)))
	if l.subtitle != "" {
		b.WriteString(fmt.Sprintf(`<div style="font-family:%s; color:#b0b0b0; font-size:13px; padding-top:4px;">%s</div>`, fontStack, html.EscapeString(l.subtitle)))
	}
	b.WriteString(`</td></tr>`)
}

func (l emailLayout) writeIntro(b *strings.Builder) {
	b.WriteString(fmt.Sprintf(`<tr><td style="padding:28px 32px 8px 32px; font-family:%s; font-size:14px; line-height:1.6; color:#333333;">`, fontStack))
	b.WriteString(l.introHTML)
	b.WriteString(`</td></tr>`)
}

func (l emailLayout) writeSummary(b *strings.Builder) {
	if l.noVulnsMsg != "" {
		b.WriteString(`<tr><td style="padding:20px 32px 8px 32px;">`)
		b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f3faf3; border:1px solid #a7d6a7; border-radius:6px;"><tr>`)
		b.WriteString(fmt.Sprintf(`<td style="padding:16px 18px; font-family:%s; font-size:14px; color:#3d7317; font-weight:bold;">&#10003;&nbsp; %s</td>`, fontStack, html.EscapeString(l.noVulnsMsg)))
		b.WriteString(`</tr></table></td></tr>`)
		return
	}
	if len(l.statCards) == 0 {
		return
	}

	b.WriteString(`<tr><td style="padding:20px 24px 8px 24px;"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0"><tr>`)
	width := 100 / len(l.statCards)
	for _, c := range l.statCards {
		b.WriteString(fmt.Sprintf(`<td width="%d%%" valign="top" style="padding:8px;">`, width))
		b.WriteString(fmt.Sprintf(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f5f5f5; border:1px solid #d2d2d2; border-radius:6px;"><tr><td style="padding:16px 18px; font-family:%s;">`, fontStack))
		b.WriteString(fmt.Sprintf(`<div style="font-size:34px; font-weight:bold; color:%s; line-height:1;">%s</div>`, colorInk, html.EscapeString(c.value)))
		b.WriteString(fmt.Sprintf(`<div style="font-size:12px; color:%s; padding-top:6px; text-transform:uppercase; letter-spacing:.5px;">%s</div>`, colorMuted, html.EscapeString(c.label)))
		b.WriteString(`</td></tr></table></td>`)
	}
	b.WriteString(`</tr></table></td></tr>`)
}

func (l emailLayout) writeSeverities(b *strings.Builder) {
	if len(l.severities) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf(`<tr><td style="padding:8px 32px 4px 32px; font-family:%s; font-size:12px; color:%s; text-transform:uppercase; letter-spacing:.5px;">CVE severity</td></tr>`, fontStack, colorMuted))
	b.WriteString(fmt.Sprintf(`<tr><td style="padding:4px 32px 20px 32px; font-family:%s;">`, fontStack))
	for _, s := range l.severities {
		color := severityColorByValue[s]
		if color == "" {
			color = colorMuted
		}
		b.WriteString(fmt.Sprintf(`<span style="display:inline-block; margin-right:20px; font-size:13px; font-weight:bold; color:%s; white-space:nowrap;"><span style="color:%s; font-size:16px; line-height:1; vertical-align:-1px;">&#9679;</span>&nbsp; %s</span>`, colorInk, color, html.EscapeString(cveSeverityToText[s])))
	}
	b.WriteString(`</td></tr>`)
}

func (l emailLayout) writeDetails(b *strings.Builder) {
	if len(l.detailRows) == 0 {
		return
	}
	b.WriteString(fmt.Sprintf(`<tr><td style="padding:0 32px 8px 32px; font-family:%s; font-size:12px; color:%s; text-transform:uppercase; letter-spacing:.5px;">Report configuration</td></tr>`, fontStack, colorMuted))
	b.WriteString(`<tr><td style="padding:0 32px 28px 32px;">`)
	b.WriteString(fmt.Sprintf(`<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="border:1px solid %s; border-radius:6px; border-collapse:separate; overflow:hidden;">`, colorBorder))

	for i, r := range l.detailRows {
		trStyle := ""
		if i%2 == 0 {
			trStyle = fmt.Sprintf(` style="background-color:%s;"`, colorRowAlt)
		}
		border := "border-bottom:1px solid #eeeeee;"
		if i == len(l.detailRows)-1 {
			border = ""
		}
		labelWidth := ""
		if i == 0 {
			labelWidth = " width:42%;"
		}

		value := html.EscapeString(r.value)
		valueFontSize := "13px"
		valueWeight := ""
		if r.bold {
			valueWeight = " font-weight:bold;"
		}
		if r.mono {
			valueFontSize = "12px"
			value = fmt.Sprintf(`<span style="font-family:'Courier New',Courier,monospace; background-color:#f0f0f0; padding:2px 6px; border-radius:3px;">%s</span>`, value)
		}

		b.WriteString(fmt.Sprintf(`<tr%s>`, trStyle))
		b.WriteString(fmt.Sprintf(`<td style="padding:12px 16px; font-family:%s; font-size:13px; color:%s;%s %s">%s</td>`, fontStack, colorMuted, labelWidth, border, html.EscapeString(r.label)))
		b.WriteString(fmt.Sprintf(`<td style="padding:12px 16px; font-family:%s; font-size:%s; color:%s;%s %s word-break:break-word;">%s</td>`, fontStack, valueFontSize, colorInk, valueWeight, border, value))
		b.WriteString(`</tr>`)
	}

	b.WriteString(`</table></td></tr>`)
}

func (l emailLayout) writeAttachmentNote(b *strings.Builder) {
	b.WriteString(`<tr><td style="padding:0 32px 28px 32px;">`)
	if l.hasAttachment {
		b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f0f7ff; border:1px solid #b3d4f5; border-radius:6px;"><tr>`)
		b.WriteString(fmt.Sprintf(`<td style="padding:14px 18px; font-family:%s; font-size:13px; color:#0066cc; line-height:1.5;">&#128206; The full CVE breakdown is attached as a CSV/ZIP file to this email.</td>`, fontStack))
	} else {
		b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background-color:#f5f5f5; border:1px solid #d2d2d2; border-radius:6px;"><tr>`)
		b.WriteString(fmt.Sprintf(`<td style="padding:14px 18px; font-family:%s; font-size:13px; color:%s; line-height:1.5;">No report file is attached because no CVEs were found.</td>`, fontStack, colorMuted))
	}
	b.WriteString(`</tr></table></td></tr>`)
}

func (l emailLayout) writeButton(b *strings.Builder) {
	if l.reportURL == "" {
		return
	}
	b.WriteString(`<tr><td align="center" style="padding:0 32px 28px 32px;">`)
	b.WriteString(`<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>`)
	b.WriteString(fmt.Sprintf(`<td align="center" style="border-radius:4px; background-color:%s;">`, colorInk))
	b.WriteString(fmt.Sprintf(`<a href="%s" style="display:inline-block; padding:12px 28px; font-family:%s; font-size:14px; font-weight:bold; color:#ffffff; text-decoration:none; border-radius:4px;">View report in console &rarr;</a>`, html.EscapeString(l.reportURL), fontStack))
	b.WriteString(`</td></tr></table></td></tr>`)
}

func (l emailLayout) writeFooter(b *strings.Builder) {
	b.WriteString(fmt.Sprintf(`<tr><td style="background-color:%s; border-top:1px solid %s; padding:18px 32px; font-family:%s; font-size:11px; color:#8a8d90; line-height:1.5;">`, colorRowAlt, colorBorder, fontStack))
	b.WriteString(`This is an automated report generated by Red Hat Advanced Cluster Security for Kubernetes. You are receiving it because you are on the mailing list for this report configuration.`)
	b.WriteString(`</td></tr>`)
}
