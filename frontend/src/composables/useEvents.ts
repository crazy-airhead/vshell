import { Events } from '@wailsio/runtime'

export function useEvents() {
  function on(event: string, callback: (ev: any) => void) {
    Events.On(event, callback)
  }

  function off(event: string) {
    Events.Off(event)
  }

  return { on, off }
}
