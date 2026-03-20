import { inject, type InjectionKey } from 'vue'

export const consoleLayoutKey: InjectionKey<any> = Symbol('console-layout')

export function useConsoleLayout() {
  const context = inject(consoleLayoutKey, null)
  if (!context) {
    throw new Error('Console layout context is not available')
  }
  return context
}
