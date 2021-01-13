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
        },
        {
          "fileName": "<FFSName>.stories.tsx",
          "template": "React Component Storybook File"
        }
      ]          
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
    ],
    "React Component Storybook File": [
      "import React from 'react';",
      "import { Meta, Story } from '@storybook/react/types-6-0';",
      "",
      "import <FFSName> from './<FFSName>';",
      "",
      "export default {",
      "    title: '<FFSName>',",
      "    component: <FFSName>,",
      "} as Meta;",
      "",
      "export const FirstStory: Story = () => <<FFSName> />;",
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

### Storybook

It's highly recommended to add [Storybook](https://storybook.js.org/) stories
for every component. Use `yarn storybook` to launch a Storybook dev server that
supports hot reloading on component changes. Storybook is loaded with light and
dark themes from the `@stackrox/tailwind-config` package.

### Linting

Use `yarn lint-fix` to reformat files (sources, CSS, JSON and markdown) in
accordance with [Prettier](https://prettier.io/) and
[ESLint](https://eslint.org/) rules defined for this package.
