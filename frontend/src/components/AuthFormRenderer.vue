<template>
  <div class="flex flex-col gap-3">
    <div v-for="field in form.fields" :key="field.name">
      <label class="block mb-1.5 text-gray-700">{{ field.label || field.name }}</label>
      <div v-if="field.type === 'TEXT'">
        <n-input v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === 'NUMBER'">
        <n-input type="number" v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === 'PASSWORD'">
        <n-input type="password" v-model:value="values[field.name]" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === 'SELECT'">
        <n-select v-model:value="values[field.name]" :options="(field.options || []).map(o => ({ label: o, value: o }))" :placeholder="field.placeholder || ''" class="w-full" />
      </div>
      <div v-else-if="field.type === 'CHECKBOX'">
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
const props = defineProps({
  form: { type: Object, required: true },
  modelValue: { type: Object, required: true }
})
const emit = defineEmits(['update:modelValue'])

const { modelValue: values } = toRefs(props)

watch(values, (v) => emit('update:modelValue', v), { deep: true })
</script>
