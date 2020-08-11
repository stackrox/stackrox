# StackRox Base Tailwind Config and CSS Themes

Base [Tailwind CSS](https://tailwindcss.com/) config recommended for all
StackRox web applications.

## Installation

Ensure dev env and project are setup by following
[these instructions](https://stack-rox.atlassian.net/wiki/spaces/ENGKB/pages/1411515467/Using+GitHub+Packages+with+NPM).

Then install the package, typically it will be a dev dependency:

```
$ yarn add @stackrox/tailwind-config --dev
```

## Usage

Common case would be to extend or use as-is the base config which is the main
export of this package. In addition, this package provides CSS styles for light
and dark themes that can be imported directly into CSS.

Alternatively, instead of importing Tailwind config and CSS themes directly,
this package offers compiled all-in-one CSS file for the use-cases that don't
require Tailwind CSS processing.

### Base Config

Import base config in your `tailwind.config.js`. Then it can be extended as a
regular JS object or re-exported as-is:

```js
const baseConfig = require('@stackrox/tailwind-config');
module.exports = baseConfig;
```

### CSS Themes

To import styles for light and dark themes import the files in accordance with
your CSS processor import syntax. E.g. if you're using
[CRA](https://create-react-app.dev/) webpack you can import them in one of your
CSS files:

```css
@import '~@stackrox/tailwind-config/light.theme.css';
@import '~@stackrox/tailwind-config/dark.theme.css';
```

Imported CSS themes add `.theme-light` and `.theme-dark` selectors. Putting
those classes on HTML container element (typically `<body/>`) will define values
for CSS variables used in the base Tailwind config.

### All-in-one CSS

Some projects may not have the need to use
[Tailwind directives](https://tailwindcss.com/docs/functions-and-directives/)
and even not have their own CSS files at all. In this case they might only
import all-in-one `tailwind.css` that contains all the Tailwind utilities
defined by the base config and the CSS themes.

## Development

There are no explicit tests nor backward compatibility checks at the moment. Yet
the `build` script will at least run the config through Tailwind.
