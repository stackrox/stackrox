# ML Risk Service Presentation

This directory contains a Quarto-based RevealJS presentation explaining the ML Risk Service training and prediction algorithms.

## Prerequisites

Install Quarto CLI:
- macOS: `brew install quarto`
- Linux: Download from https://quarto.org/docs/get-started/
- Windows: Download installer from https://quarto.org/docs/get-started/

## Rendering the Presentation

### Generate HTML slides

```bash
cd presentation
quarto render index.qmd
```

This creates `index.html` in the current directory.

### Preview with live reload

```bash
quarto preview index.qmd
```

Opens browser at http://localhost:XXXX with live reload on file changes.

### Render to PDF

```bash
quarto render index.qmd --to revealjs-pdf
```

Requires Chrome/Chromium installed.

### Render to PowerPoint

```bash
quarto render index.qmd --to pptx
```

## Presentation Controls

- **Arrow keys** or **Space**: Navigate slides
- **F**: Enter fullscreen
- **S**: Open speaker notes view
- **O**: Overview mode (see all slides)
- **ESC**: Exit fullscreen/overview

## File Structure

```
presentation/
├── README.md              # This file
├── index.qmd              # Main slide deck (Quarto markdown)
├── styles.css             # Custom RevealJS styling
└── diagrams/              # Technical SVG diagrams
    ├── randomforest-architecture.svg
    ├── training-pipeline.svg
    ├── prediction-flow.svg
    └── feature-categories.svg
```

## Customization

### Modify slides

Edit `index.qmd` using standard Quarto markdown syntax.

### Adjust styling

Edit `styles.css` for custom CSS rules.

### Change theme

Update `format.revealjs.theme` in index.qmd YAML header:
- Available themes: `simple`, `dark`, `league`, `sky`, `beige`, `serif`, `night`, `moon`, `solarized`

## Presentation Details

- **Duration**: ~20 minutes
- **Audience**: ACS Frankfurt Face-to-Face 2025
- **Topics**: ML training pipeline, RandomForest algorithm, prediction workflow, feature engineering
- **Diagrams**: 4 technical SVG visualizations

## Troubleshooting

### Quarto not found
Ensure Quarto is in your PATH after installation.

### Diagrams not displaying
Check that SVG files exist in `diagrams/` directory with correct paths in index.qmd.

### Live preview not updating
Try stopping preview (Ctrl+C) and restarting `quarto preview index.qmd`.

## Resources

- [Quarto Documentation](https://quarto.org/docs/)
- [RevealJS Documentation](https://revealjs.com/)
- [Quarto RevealJS Features](https://quarto.org/docs/presentations/revealjs/)
