<template>
  <div class="flex flex-col gap-3">
    <div v-for="field in form.fields" :key="field.name">
      <label class="block mb-1.5 text-gray-700">{{ field.label || field.name }}</label>
      <div v-if="field.type === FieldType.TEXT">
        <n-input v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === FieldType.NUMBER">
        <n-input type="number" v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === FieldType.PASSWORD">
        <n-input type="password" show-password-on="click" v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === FieldType.SELECT">
        <n-select v-model:value="values[field.name]" :options="(field.options || []).map(o => ({ label: o, value: o }))" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === FieldType.CHECKBOX">
        <n-checkbox v-model:value="values[field.name]">{{ field.label || field.name }}</n-checkbox>
      </div>
      <div v-else>
        <n-input v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { toRefs, watch } from 'vue'

// FieldType mirrors PluginV1_AuthField_FieldType enum (proto int values)
const FieldType = { TEXT: 1, NUMBER: 2, PASSWORD: 3, CHECKBOX: 4, SELECT: 5 }

const props = defineProps({
  form: { type: Object, required: true },
  modelValue: { type: Object, required: true }
})
const emit = defineEmits(['update:modelValue'])

const { modelValue: values } = toRefs(props)

watch(values, (v) => emit('update:modelValue', v), { deep: true })
</script>
