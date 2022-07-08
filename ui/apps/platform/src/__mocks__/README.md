This directory holds mocks used for Jest [manual mocking](https://jestjs.io/docs/manual-mocks#using-with-es-module-imports) in order to 
mock `node_module` dependencies in a central location.

Note that the Jest docs specify that this directory [should be placed adjacent](https://jestjs.io/docs/manual-mocks#mocking-node-modules) to 
the `node_modules` directory, but this is not correct when using CRA/react-scripts. Based on discussion in [this issue](https://github.com/facebook/create-react-app/issues/7539) 
the documented behavior was broken, and mocks will only automatically be picked up if they reside under the `src` directory. This breakage still appears to exist in 
the latest version of [CRA](https://github.com/facebook/create-react-app/blob/main/packages/react-scripts/scripts/utils/createJestConfig.js#L26).

Also of note is that ESM `export` does not appear to work in these manual mocks. e.g.
```typescript
// works
module.exports = myMockedModule;
// does not work, resulting in all imports in dependent modules being `undefined`
export default myMockedModule;
```