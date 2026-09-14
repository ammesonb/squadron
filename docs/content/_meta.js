const hidden = { display: 'hidden' }

export default {
  // These legacy root routes remain available to direct links and search, but
  // should not bleed into every section's sidebar. The primary documentation
  // areas below each own their local navigation tree.
  index: hidden,
  'declarative-agent-framework': hidden,
  'getting-started': hidden,

  product: {
    type: 'page',
    title: 'Product',
  },
  config: {
    type: 'page',
    title: 'Configuration',
  },
  guides: {
    type: 'page',
    title: 'Guides',
  },

  compare: hidden,
  faq: hidden,
}
