import antfu from '@antfu/eslint-config'

export default antfu({
  ignores: [
    '**/bindings/**/*',
  ],
  typescript: true,
  vue: true,
  rules: {
    'vue/custom-event-name-casing': ['error', 'kebab-case'],
    '@stylistic/max-statements-per-line': ['error', { max: 2 }],
    'no-console': ['error', {
      allow: [
        'warn',
        'error',
        'debug',
        'info',
        'log',
      ],
    }],
  },
})
