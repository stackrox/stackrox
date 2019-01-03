module.exports = function (variants) {
  return function ({ addUtilities }) {
    addUtilities({
      '.object-contain': { objectFit: 'contain' },
      '.object-cover': { objectFit: 'cover' },
      '.object-fill': { objectFit: 'fill' },
      '.object-none': { objectFit: 'none' },
      '.object-scale': { objectFit: 'scale-down' },
    }, variants)
  }
}