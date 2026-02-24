<template>
  <n-modal
    v-model:show="localVisible"
    :mask-closable="true"
    @after-leave="reset"
  >
    <n-card
      :title="formConfig.title"
      style="max-width: 440px; width: 90vw"
      :bordered="false"
      role="dialog"
      aria-modal="true"
    >
      <n-form @submit.prevent="submit">
        <n-form-item
          v-for="field in formConfig.fields"
          :key="field.key"
          :label="field.label"
        >
          <n-input
            v-model:value="formValues[field.key]"
            :placeholder="field.placeholder"
            :autofocus="field === formConfig.fields[0]"
            @keydown.enter.prevent="submit"
          />
        </n-form-item>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2 pt-1">
          <n-button @click="cancel">Cancel</n-button>
          <n-button
            type="primary"
            :disabled="!isValid"
            @click="submit"
          >
            Execute
          </n-button>
        </div>
      </template>
    </n-card>
  </n-modal>
</template>

<script setup>
import { ref, computed, watch } from "vue"

const props = defineProps({
  visible: { type: Boolean, default: false },
  /** ConnectionTreeAction object: { type, title, query } */
  action: { type: Object, default: null },
})

const emit = defineEmits(["update:visible", "submit", "cancel"])

// ---------- derived form config ------------------------------------------

/**
 * Returns a form configuration object for the given action.
 * Shape: { title: string, fields: Field[], buildQuery: (rawQuery, vals) => string }
 * Field shape: { key, label, placeholder, default }
 */
function getFormConfig(action) {
  if (!action) return { title: "", fields: [], buildQuery: (q) => q }

  if (action.type === "create-database") {
    // SQLite ATTACH DATABASE needs a path + alias rather than a simple name.
    if (/ATTACH\s+DATABASE/i.test(action.query ?? "")) {
      return {
        title: action.title ?? "Attach Database",
        fields: [
          {
            key: "path",
            label: "File path",
            placeholder: "/path/to/database.db",
            default: "",
          },
          {
            key: "alias",
            label: "Alias",
            placeholder: "other_db",
            default: "other",
          },
        ],
        buildQuery: (_q, vals) =>
          `ATTACH DATABASE '${vals.path}' AS ${vals.alias};`,
      }
    }

    return {
      title: action.title ?? "Create Database",
      fields: [
        {
          key: "name",
          label: "Database name",
          placeholder: "my_database",
          default: "new_database",
        },
      ],
      buildQuery: (q, vals) =>
        q.replace(/new_database|new_collection/gi, vals.name),
    }
  }

  if (action.type === "create-table") {
    const isCollection = /CREATE\s+COLLECTION/i.test(action.query ?? "")
    return {
      title: action.title ?? (isCollection ? "Create Collection" : "Create Table"),
      fields: [
        {
          key: "name",
          label: isCollection ? "Collection name" : "Table name",
          placeholder: isCollection ? "my_collection" : "my_table",
          default: isCollection ? "new_collection" : "new_table",
        },
      ],
      buildQuery: (q, vals) =>
        q.replace(/new_table|new_collection/gi, vals.name),
    }
  }

  return { title: action.title ?? "", fields: [], buildQuery: (q) => q }
}

// ---------- reactive state -----------------------------------------------

const localVisible = computed({
  get: () => props.visible,
  set: (v) => emit("update:visible", v),
})

const formConfig = computed(() => getFormConfig(props.action))
const formValues = ref({})

// Re-initialise form values whenever the incoming action changes.
watch(
  () => props.action,
  (newAction) => {
    if (!newAction) return
    const config = getFormConfig(newAction)
    formValues.value = Object.fromEntries(
      config.fields.map((f) => [f.key, f.default ?? ""]),
    )
  },
  { immediate: true },
)

const isValid = computed(() =>
  formConfig.value.fields.every(
    (f) => (formValues.value[f.key] ?? "").trim() !== "",
  ),
)

// ---------- actions -------------------------------------------------------

function submit() {
  if (!isValid.value) return
  const modifiedQuery = formConfig.value.buildQuery(
    props.action?.query ?? "",
    formValues.value,
  )
  emit("submit", modifiedQuery)
  localVisible.value = false
}

function cancel() {
  emit("cancel")
  localVisible.value = false
}

function reset() {
  formValues.value = {}
}
</script>
