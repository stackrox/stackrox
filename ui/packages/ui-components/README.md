**Attention:** Now that StackRox is a part of Red Hat, we are migrating to Red
Hat's design system, [PatternFly](https://www.patternfly.org/). We will be using
PatternFly's React components directly as a rule, and no longer adding to this
package, or updating the components already here.

---

# StackRox React UI Components

A library of React components recommended for any StackRox UI for the
consistency and reusability.

## Installation

Ensure dev env and project are setup by following
[these instructions](https://stack-rox.atlassian.net/wiki/spaces/ENGKB/pages/1411515467/Using+GitHub+Packages+with+NPM).

Then install the package

```
$ yarn add @stackrox/ui-components
```

## Usage

Import components as ES6 modules

```js
import { Tooltip } from '@stackrox/ui-components';
```

Every `{Component}` also exports its propTypes as a TypeScript type definition
as a named export `{Component}Props`. This type can be imported directly from
the component's module:

```js
import { AvatarProps } from '@stackrox/ui-components/lib/Avatar';
```

To include default styling somewhere in your CSS or JS import
`lib/ui-components.css`:

```css
@import '~@stackrox/ui-components/lib/ui-components.css';
```

Component styles depend on CSS variables and Tailwind CSS class names. It's
strongly encouraged that you use the themes from `@stackrox/tailwind-config`
with these components. If not, you will need to recreate ALL the variables
defined there in your own custom CSS theme to make these components usable. (In
other words, just use `@stackrox/tailwind-config`.)

_Note: currently the package is built with CommonJS. Therefore, when importing
components as ES6 modules, bundlers like [webpack](https://webpack.js.org/) or
[Rollup](https://rollupjs.org/guide/en/) will include the whole package into the
final build. It'll be changed in the future versions of this library with `es6`
build provided as well. Therefore it's still recommended to import components as
ES6 modules to benefit from it in the near future._

## Development

### IDE

While the project is IDE agnostic, there are some
[VS Code](https://code.visualstudio.com/) extensions that can ease the
development.
[Fast Folder Structure (FFS)](https://marketplace.visualstudio.com/items?itemName=Huuums.vscode-fast-folder-structure)
is in particular useful for a quick creation of component directory structure.
After installing it, add the following configuration to your `settings.json`
file:

<details>
  <summary>Expand to see FFS configuration...</summary>

```json
"fastFolderStructure.structures": [
  {
    "name": "TypeScript React Component Dir",
    "omitParentDirectory": false,
    "structure": [
      {
        "fileName": "<FFSName>.tsx",
        "template": "React TypeScript Functional Component with PropTypes"
      },
      {
        "fileName": "index.ts",
        "template": "React Component Index File"
      },
      {
        "fileName": "<FFSName>.test.tsx",
        "template": "React Component Jest Tests"
      }    ]
  }
],
"fastFolderStructure.fileTemplates": {
  "React TypeScript Functional Component with PropTypes": [
    "import React, { ReactElement } from 'react';",
    "import PropTypes, { InferProps } from 'prop-types';",
    "",
    "function <FFSName>({}: <FFSName>Props): ReactElement {",
    "    return <></>;",
    "}",
    "",
    "<FFSName>.propTypes = {};",
    "",
    "<FFSName>.defaultProps = {};",
    "",
    "export type <FFSName>Props = InferProps<typeof <FFSName>.propTypes>;",
    "export default <FFSName>;",
    ""
  ],
  "React Component Index File": [
    "export { default } from './<FFSName>';",
    "export * from './<FFSName>';",
    ""
  ],
  "React Component Jest Tests": [
    "import React from 'react';",
    "import { render } from '@testing-library/react';",
    "",
    "import <FFSName> from './<FFSName>';",
    "",
    "describe('<FFSName>', () => {",
    "    test('renders title, subtitle and footer', () => {",
    "        const { getByText, getByTestId } = render(<<FFSName> />);",
    "    });",
    "});",
    ""
  ]
}
```

</details>

Then in the project explorer's context menu for `src` directory select
`FFS: Create new Folder` and enter the component name to add to the library.

### Testing

[Jest](https://jestjs.io/) and
[React Testing Library](https://testing-library.com/docs/react-testing-library/intro)
are used for component unit testing. It's highly recommended to add tests for
every component. Those tests at minimum should validate that component can be
successfully rendered.

### Linting

Use `yarn lint-fix` to reformat files (sources, CSS, JSON and markdown) in
accordance with [Prettier](https://prettier.io/) and
[ESLint](https://eslint.org/) rules defined for this package.
