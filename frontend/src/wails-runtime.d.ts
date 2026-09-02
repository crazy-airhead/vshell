declare module '@wailsio/runtime' {
  export const Events: {
    On(eventName: string, callback: (ev: any) => void): () => void
    OnMultiple(eventName: string, callback: (ev: any) => void, maxCallbacks: number): () => void
    Once(eventName: string, callback: (ev: any) => void): () => void
    Off(...eventNames: [string, ...string[]]): void
    OffAll(): void
    Emit(name: string, data?: any): Promise<boolean>
  }
  export const Call: {
    ByID(id: number, ...args: any[]): Promise<any>
    ByName(name: string, ...args: any[]): Promise<any>
  }
  export const Clipboard: {
    SetText(text: string): Promise<void>
    Text(): Promise<string>
  }
}
