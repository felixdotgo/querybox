<script setup>
import { toRefs, watch } from 'vue'
import { OpenFileDialog } from '../../bindings/github.com/felixdotgo/querybox/services/app.js'

const props = defineProps({
  form: { type: Object, required: true },
  modelValue: { type: Object, required: true },
})

const emit = defineEmits(['update:modelValue'])

// FieldType mirrors PluginV1_AuthField_FieldType enum (proto int values)
const FieldType = { TEXT: 1, NUMBER: 2, PASSWORD: 3, CHECKBOX: 4, SELECT: 5, FILE_PATH: 6 }

const { modelValue: values } = toRefs(props)

watch(values, v => emit('update:modelValue', v), { deep: true })

async function pickFile(fieldName) {
  const path = await OpenFileDialog()
  if (path) {
    values.value[fieldName] = path
  }
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div v-for="field in form.fields" :key="field.name">
      <label class="block mb-1.5 text-gray-700">{{ field.label || field.name }}</label>
      <div v-if="field.type === FieldType.TEXT">
        <n-input v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === FieldType.NUMBER">
        <n-input v-model:value="values[field.name]" type="number" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === FieldType.PASSWORD">
        <n-input v-model:value="values[field.name]" type="password" show-password-on="click" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === FieldType.SELECT">
        <n-select v-model:value="values[field.name]" :options="(field.options || []).map(o => ({ label: o, value: o }))" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === FieldType.CHECKBOX">
        <n-checkbox v-model:value="values[field.name]">
          {{ field.label || field.name }}
        </n-checkbox>
      </div>
      <div v-else-if="field.type === FieldType.FILE_PATH" class="flex gap-2">
        <n-input v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="flex-1" />
        <n-button title="Browse for file" @click="pickFile(field.name)">
          <template #icon>
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
            </svg>
          </template>
        </n-button>
      </div>
      <div v-else>
        <n-input v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
    </div>
  </div>
</template>
