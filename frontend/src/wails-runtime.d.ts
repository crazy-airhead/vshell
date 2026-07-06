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
  export const Dialogs: {
    OpenFile(options: {
      Title?: string
      Message?: string
      ButtonText?: string
      Directory?: string
      CanChooseFiles?: boolean
      CanChooseDirectories?: boolean
      AllowsMultipleSelection?: boolean
      Filters?: Array<{ DisplayName?: string; Pattern?: string }>
    }): Promise<string | string[]>
    SaveFile(options: {
      Title?: string
      Message?: string
      ButtonText?: string
      Directory?: string
      Filename?: string
      Filters?: Array<{ DisplayName?: string; Pattern?: string }>
    }): Promise<string>
  }
}
