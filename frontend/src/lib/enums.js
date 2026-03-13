/** Plugin type enum matching PluginV1.Type protobuf enum values. */
export const PluginType = Object.freeze({
  DRIVER: 1,
  TRANSFORMER: 2,
  FORMATTER: 3,
})

/** Human-readable labels for plugin types. */
export const PLUGIN_TYPE_LABELS = Object.freeze({
  [PluginType.DRIVER]: 'Driver',
  [PluginType.TRANSFORMER]: 'Transformer',
  [PluginType.FORMATTER]: 'Formatter',
})
