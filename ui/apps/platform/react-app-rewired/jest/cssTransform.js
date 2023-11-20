'use strict';

// This is a custom Jest transformer turning style imports into empty objects.
// http://facebook.github.io/jest/docs/en/webpack.html

module.exports = {
  process() {
    // See config-overrides.js file
    // See node_modules/react-scripts/config/jest/cssTransform.js
    // Return object instead of string.
    return {
      code: 'module.exports = {};',
    };
  },
  getCacheKey() {
    // The output is always the same.
    return 'cssTransform';
  },
};
