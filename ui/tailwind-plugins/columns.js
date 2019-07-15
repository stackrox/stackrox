const _ = require("lodash");

module.exports = function({ index = {}, variants = ["responsive"] }) {
  return function({ e, addUtilities }) {
    addUtilities(
      [
        ..._.map(index, (value, name) => ({
          [`.${e(`columns-${name}`)}`]: { columnCount: value },
          [`.${e(`columns-gap-${name}`)}`]: { columnGap: value }
        }))
      ],
      variants
    );
  };
};
