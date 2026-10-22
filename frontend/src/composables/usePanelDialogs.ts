import { h, ref } from 'vue'
import { NInput, useDialog } from 'naive-ui'
import { t } from '../i18n'

export function usePanelDialogs() {
  const dialog = useDialog()
  function confirm(content: string): Promise<boolean> {
    return new Promise(resolve => {
      dialog.warning({
        title: t('m_ab190f1faa64'), content,
        positiveText: t('m_a3ea3c17b401'), negativeText: t('m_2cd0f3be8738'),
        onPositiveClick: () => resolve(true), onAfterLeave: () => resolve(false),
      })
    })
  }
  function prompt(title: string, initial = '', password = false): Promise<string | null> {
    const value = ref(initial)
    return new Promise(resolve => {
      dialog.create({
        title,
        content: () => h(NInput, { value: value.value, autofocus: true,
          type: password ? 'password' : 'text', 'onUpdate:value': next => { value.value = next } }),
        positiveText: t('m_a3ea3c17b401'), negativeText: t('m_2cd0f3be8738'),
        onPositiveClick: () => resolve(value.value), onAfterLeave: () => resolve(null),
      })
    })
  }
  return { confirm, prompt }
}
