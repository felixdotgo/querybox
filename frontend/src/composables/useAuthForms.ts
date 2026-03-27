import { ref, watch, type Ref } from 'vue'
import { GetPluginAuthForms } from '@/bindings/github.com/felixdotgo/querybox/services/pluginmgr/manager'
import type { AuthForm, SavedCredential } from '@/lib/types'

/**
 * Shared auth-form state management used by both the "create connection" and
 * "edit connection" views.
 */
export function useAuthForms() {
  const authForms: Ref<Record<string, AuthForm>> = ref({})
  const selectedAuthForm: Ref<string> = ref('')
  const authValues: Ref<Record<string, string>> = ref({})

  function resetAuthState(): void {
    authForms.value = {}
    selectedAuthForm.value = ''
    authValues.value = {}
  }

  /**
   * Initialize field values for the currently selected auth form tab,
   * applying defaults only for fields the user hasn't typed a value for yet.
   */
  function initFieldDefaults(formKey: string): void {
    const def = authForms.value[formKey]
    if (!def)
      return
    for (const f of def.fields || []) {
      if (!f) continue
      if (authValues.value[f.name!] === undefined || authValues.value[f.name!] === null) {
        authValues.value[f.name!] = f.value ?? ''
      }
    }
  }

  // Keep authValues in sync when the user switches auth form tabs.
  watch(selectedAuthForm, (newKey: string) => {
    if (!newKey)
      return
    initFieldDefaults(newKey)
  })

  /**
   * Load auth forms for a given driver and optionally pre-fill with saved
   * credential values.
   */
  async function loadAuthForms(driverType: string, saved?: SavedCredential): Promise<boolean> {
    resetAuthState()
    try {
      const resp = await GetPluginAuthForms(driverType)
      if (!resp || Object.keys(resp).length === 0)
        return false

      authForms.value = resp as Record<string, AuthForm>
      const formKeys = Object.keys(authForms.value)

      // Select the appropriate form tab
      if (saved?.form && authForms.value[saved.form]) {
        selectedAuthForm.value = saved.form
      }
      else {
        selectedAuthForm.value = formKeys[0]
      }

      // Initialize field defaults
      authValues.value = {}
      initFieldDefaults(selectedAuthForm.value)

      // Overwrite with saved values if provided
      if (saved?.values) {
        Object.assign(authValues.value, saved.values)
      }

      return true
    }
    catch {
      return false
    }
  }

  /**
   * Serialize the current auth form state into a credential blob string.
   * Returns empty string if no auth forms are active.
   */
  function serializeCredential(): string {
    if (Object.keys(authForms.value || {}).length === 0)
      return ''
    return JSON.stringify({ form: selectedAuthForm.value, values: authValues.value })
  }

  return {
    authForms,
    selectedAuthForm,
    authValues,
    resetAuthState,
    loadAuthForms,
    serializeCredential,
  }
}
